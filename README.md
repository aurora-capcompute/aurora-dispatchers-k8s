# aurora-k8s

Native Kubernetes dispatcher for Aurora.

Each operation is an independent capability enabled in the thread manifest:

```json
{
  "capabilities": [
    {"name": "k8s.get", "settings": {"namespaces": ["default"]}},
    {"name": "k8s.list", "settings": {"namespaces": ["default"]}},
    {"name": "k8s.logs", "settings": {"namespaces": ["default"]}},
    {"name": "k8s.events", "settings": {}},
    {"name": "k8s.apply", "settings": {"namespaces": ["default"]}},
    {"name": "k8s.delete", "settings": {"namespaces": ["default"]}}
  ]
}
```

Mutating operations (`k8s.apply`, `k8s.delete`) require human approval by
default. Set `"require_approval": false` to disable, or `"require_approval":
true` on read operations to enable.

Settings per capability:

- `kubeconfig`: path to kubeconfig (optional, defaults to in-cluster or `~/.kube/config`).
- `context`: kubeconfig context (optional).
- `namespaces`: allowed namespaces (empty = all).
- `require_approval`: override the default approval behavior.
