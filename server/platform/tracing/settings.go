package tracing

import "time"

// Settings is the runtime tunable head sampling configuration. The settings live in the trace_sampler_settings singleton row
// and are polled by each Fleet replica so support can adjust sampling without a restart.
type Settings struct {
	HighVolumeRatio float64 `json:"high_volume_ratio" db:"high_volume_ratio"`
	StandardRatio   float64 `json:"standard_ratio" db:"standard_ratio"`
	ForceFull       bool    `json:"force_full" db:"force_full"`
	// UpdatedAt uses omitzero so the PATCH handler can zero it before echoing the response.
	UpdatedAt time.Time `json:"updated_at,omitzero" db:"updated_at"`
}
