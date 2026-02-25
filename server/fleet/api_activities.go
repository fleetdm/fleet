package fleet

import "net/http"

type ListActivitiesResponse struct {
	Meta       *PaginationMetadata `json:"meta"`
	Activities []*Activity         `json:"activities"`
	Err        error               `json:"error,omitempty"`
}

func (r ListActivitiesResponse) Error() error { return r.Err }

type ListHostUpcomingActivitiesRequest struct {
	HostID      uint        `url:"id"`
	ListOptions ListOptions `url:"list_options"`
}

type ListHostUpcomingActivitiesResponse struct {
	Meta       *PaginationMetadata `json:"meta"`
	Activities []*UpcomingActivity `json:"activities"`
	Count      uint                `json:"count"`
	Err        error               `json:"error,omitempty"`
}

func (r ListHostUpcomingActivitiesResponse) Error() error { return r.Err }

type CancelHostUpcomingActivityRequest struct {
	HostID     uint   `url:"id"`
	ActivityID string `url:"activity_id"`
}

type CancelHostUpcomingActivityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r CancelHostUpcomingActivityResponse) Error() error { return r.Err }

func (r CancelHostUpcomingActivityResponse) Status() int { return http.StatusNoContent }
