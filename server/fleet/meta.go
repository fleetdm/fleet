package fleet

type ObjectMetadata struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
}
