package fleet

type ObjectMetadata struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

type PaginationMetadata struct {
	HasNextResults     bool `json:"has_next_results"`
	HasPreviousResults bool `json:"has_previous_results"`
	// TotalResults is the total number of results found for the query (as opposed to the number
	// of results returned in the current paginated response). This field is not always set so callers
	// must take care to confirm whether a non-zero value should be expected in their specific use cases.
	TotalResults uint `json:"-"`
}
