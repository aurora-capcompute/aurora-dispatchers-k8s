package k8s

import (
	"fmt"
	"strings"
)

type Settings struct {
	Kubeconfig      string   `json:"kubeconfig,omitempty"`
	Context         string   `json:"context,omitempty"`
	Namespaces      []string `json:"namespaces,omitempty"`
	RequireApproval *bool    `json:"require_approval,omitempty"`
}

func isMutating(name string) bool {
	return name == "k8s.apply" || name == "k8s.delete"
}

func requiresApproval(name string, settings Settings) bool {
	if settings.RequireApproval != nil {
		return *settings.RequireApproval
	}
	return isMutating(name)
}

type namespacePolicy struct {
	allowed map[string]struct{}
}

func newNamespacePolicy(namespaces []string) namespacePolicy {
	allowed := make(map[string]struct{}, len(namespaces))
	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)
		if ns != "" {
			allowed[ns] = struct{}{}
		}
	}
	return namespacePolicy{allowed: allowed}
}

func (p namespacePolicy) check(namespace string, namespaced bool) error {
	if len(p.allowed) == 0 {
		return nil
	}
	if !namespaced {
		return nil
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required (allowed: %s)", p.list())
	}
	if _, ok := p.allowed[namespace]; !ok {
		return fmt.Errorf("namespace %q is not allowed (allowed: %s)", namespace, p.list())
	}
	return nil
}

func (p namespacePolicy) list() string {
	names := make([]string, 0, len(p.allowed))
	for ns := range p.allowed {
		names = append(names, ns)
	}
	return strings.Join(names, ", ")
}
