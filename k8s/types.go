package k8s

import "encoding/json"

type GetRequest struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type GetResponse struct {
	Resource json.RawMessage `json:"resource"`
}

type ListRequest struct {
	APIVersion    string `json:"api_version"`
	Kind          string `json:"kind"`
	Namespace     string `json:"namespace,omitempty"`
	LabelSelector string `json:"label_selector,omitempty"`
	FieldSelector string `json:"field_selector,omitempty"`
	Limit         int64  `json:"limit,omitempty"`
}

type ListResponse struct {
	Items []json.RawMessage `json:"items"`
	Count int               `json:"count"`
}

type ApplyRequest struct {
	Resource json.RawMessage `json:"resource"`
}

type ApplyResponse struct {
	Resource json.RawMessage `json:"resource"`
	Action   string          `json:"action"`
}

type DeleteRequest struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

type DeleteResponse struct {
	Deleted bool   `json:"deleted"`
	Name    string `json:"name"`
}

type LogsRequest struct {
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	Container  string `json:"container,omitempty"`
	TailLines  *int64 `json:"tail_lines,omitempty"`
	LimitBytes *int64 `json:"limit_bytes,omitempty"`
}

type LogsResponse struct {
	Logs string `json:"logs"`
}

type EventsRequest struct {
	Namespace      string `json:"namespace,omitempty"`
	InvolvedObject string `json:"involved_object,omitempty"`
	FieldSelector  string `json:"field_selector,omitempty"`
	Limit          int64  `json:"limit,omitempty"`
}

type EventsResponse struct {
	Items []json.RawMessage `json:"items"`
	Count int               `json:"count"`
}
