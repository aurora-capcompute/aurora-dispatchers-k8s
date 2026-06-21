package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type Client interface {
	Get(ctx context.Context, req GetRequest) (*unstructured.Unstructured, error)
	List(ctx context.Context, req ListRequest) (*unstructured.UnstructuredList, error)
	Apply(ctx context.Context, req ApplyRequest) (*unstructured.Unstructured, string, error)
	Delete(ctx context.Context, req DeleteRequest) error
	Logs(ctx context.Context, req LogsRequest) (string, error)
	Events(ctx context.Context, req EventsRequest) (*unstructured.UnstructuredList, error)
}

type client struct {
	dynamic   dynamic.Interface
	clientset kubernetes.Interface
	mapper    *restmapper.DeferredDiscoveryRESTMapper
}

func NewClient(kubeconfig, context string) (Client, error) {
	config, err := buildConfig(kubeconfig, context)
	if err != nil {
		return nil, fmt.Errorf("build k8s config: %w", err)
	}
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	disc := discovery.NewDiscoveryClientForConfigOrDie(config)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		&cachedDiscovery{DiscoveryInterface: disc},
	)
	return &client{dynamic: dyn, clientset: cs, mapper: mapper}, nil
}

func buildConfig(kubeconfig, ctx string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
			&clientcmd.ConfigOverrides{CurrentContext: ctx},
		).ClientConfig()
	}
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: ctx},
	).ClientConfig()
}

func (c *client) resolve(apiVersion, kind string) (schema.GroupVersionResource, bool, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("parse api version %q: %w", apiVersion, err)
	}
	mapping, err := c.mapper.RESTMapping(gv.WithKind(kind).GroupKind(), gv.Version)
	if err != nil {
		return schema.GroupVersionResource{}, false, fmt.Errorf("resolve %s/%s: %w", apiVersion, kind, err)
	}
	namespaced := mapping.Scope.Name() == "namespace"
	return mapping.Resource, namespaced, nil
}

func (c *client) resource(gvr schema.GroupVersionResource, namespace string, namespaced bool) dynamic.ResourceInterface {
	if namespaced && namespace != "" {
		return c.dynamic.Resource(gvr).Namespace(namespace)
	}
	return c.dynamic.Resource(gvr)
}

func (c *client) Get(ctx context.Context, req GetRequest) (*unstructured.Unstructured, error) {
	gvr, namespaced, err := c.resolve(req.APIVersion, req.Kind)
	if err != nil {
		return nil, err
	}
	return c.resource(gvr, req.Namespace, namespaced).Get(ctx, req.Name, metav1.GetOptions{})
}

func (c *client) List(ctx context.Context, req ListRequest) (*unstructured.UnstructuredList, error) {
	gvr, namespaced, err := c.resolve(req.APIVersion, req.Kind)
	if err != nil {
		return nil, err
	}
	opts := metav1.ListOptions{
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
	}
	return c.resource(gvr, req.Namespace, namespaced).List(ctx, opts)
}

func (c *client) Apply(ctx context.Context, req ApplyRequest) (*unstructured.Unstructured, string, error) {
	var obj unstructured.Unstructured
	if err := obj.UnmarshalJSON(req.Resource); err != nil {
		return nil, "", fmt.Errorf("decode resource: %w", err)
	}
	apiVersion := obj.GetAPIVersion()
	kind := obj.GetKind()
	name := obj.GetName()
	namespace := obj.GetNamespace()
	if apiVersion == "" || kind == "" || name == "" {
		return nil, "", fmt.Errorf("resource must have apiVersion, kind, and metadata.name")
	}
	gvr, namespaced, err := c.resolve(apiVersion, kind)
	if err != nil {
		return nil, "", err
	}

	res := c.resource(gvr, namespace, namespaced)
	_, getErr := res.Get(ctx, name, metav1.GetOptions{})
	action := "created"
	if getErr == nil {
		action = "configured"
	}

	applied, err := res.Patch(ctx, name, types.ApplyPatchType, req.Resource, metav1.PatchOptions{
		FieldManager: "aurora",
	})
	if err != nil {
		return nil, "", fmt.Errorf("apply %s/%s: %w", kind, name, err)
	}
	return applied, action, nil
}

func (c *client) Delete(ctx context.Context, req DeleteRequest) error {
	gvr, namespaced, err := c.resolve(req.APIVersion, req.Kind)
	if err != nil {
		return err
	}
	return c.resource(gvr, req.Namespace, namespaced).Delete(ctx, req.Name, metav1.DeleteOptions{})
}

func (c *client) Logs(ctx context.Context, req LogsRequest) (string, error) {
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}
	opts := &corev1.PodLogOptions{
		Container: req.Container,
	}
	if req.TailLines != nil {
		opts.TailLines = req.TailLines
	}
	if req.LimitBytes != nil {
		opts.LimitBytes = req.LimitBytes
	}
	stream, err := c.clientset.CoreV1().Pods(namespace).GetLogs(req.Name, opts).Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("stream logs for %s/%s: %w", namespace, req.Name, err)
	}
	defer stream.Close()
	raw, err := io.ReadAll(io.LimitReader(stream, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read logs: %w", err)
	}
	return string(raw), nil
}

func (c *client) Events(ctx context.Context, req EventsRequest) (*unstructured.UnstructuredList, error) {
	selectors := []string{}
	if req.FieldSelector != "" {
		selectors = append(selectors, req.FieldSelector)
	}
	if req.InvolvedObject != "" {
		selectors = append(selectors, "involvedObject.name="+req.InvolvedObject)
	}
	return c.List(ctx, ListRequest{
		APIVersion:    "v1",
		Kind:          "Event",
		Namespace:     req.Namespace,
		FieldSelector: strings.Join(selectors, ","),
		Limit:         req.Limit,
	})
}

type cachedDiscovery struct {
	discovery.DiscoveryInterface
}

func (d *cachedDiscovery) Fresh() bool { return false }
func (d *cachedDiscovery) Invalidate() {}
func (d *cachedDiscovery) ServerGroups() (*metav1.APIGroupList, error) {
	return d.DiscoveryInterface.ServerGroups()
}
