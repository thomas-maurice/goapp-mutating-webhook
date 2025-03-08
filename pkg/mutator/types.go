package mutator

type Patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

func maybe[T any](v T) *T { return &v }
