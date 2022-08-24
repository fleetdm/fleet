package msrc_io

import "time"

const (
	msrcMinYear = 2020
	msrcBaseURL = "https://api.msrc.microsoft.com/cvrf/v2.0/document/2022-May"
)

// downloadFeed downloads the msrc security feed based on the provided month and year.
// Will error out if the the year is outside the supported range or
// if some I/O error occurs. Returns the path where the feed was downloaded.
func downloadFeed(month time.Month, year int) (string, error) {
	panic("not implemented")
}
