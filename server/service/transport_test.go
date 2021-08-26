package service

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestListOptionsFromRequest(t *testing.T) {
	var listOptionsTests = []struct {
		// url string to parse
		url string
		// expected list options
		listOptions fleet.ListOptions
		// should cause an error
		shouldErr bool
	}{
		// both params provided
		{
			url:         "/foo?page=1&per_page=10",
			listOptions: fleet.ListOptions{Page: 1, PerPage: 10},
		},
		// only per_page (page should default to 0)
		{
			url:         "/foo?per_page=10",
			listOptions: fleet.ListOptions{Page: 0, PerPage: 10},
		},
		// only page (per_page should default to defaultPerPage
		{
			url:         "/foo?page=10",
			listOptions: fleet.ListOptions{Page: 10, PerPage: defaultPerPage},
		},
		// no params provided (defaults to empty ListOptions indicating
		// unlimited)
		{
			url:         "/foo?unrelated=foo",
			listOptions: fleet.ListOptions{},
		},

		// Both order params provided
		{
			url:         "/foo?order_key=foo&order_direction=desc",
			listOptions: fleet.ListOptions{OrderKey: "foo", OrderDirection: fleet.OrderDescending},
		},
		// Both order params provided (asc)
		{
			url:         "/foo?order_key=bar&order_direction=asc",
			listOptions: fleet.ListOptions{OrderKey: "bar", OrderDirection: fleet.OrderAscending},
		},
		// Default order direction
		{
			url:         "/foo?order_key=foo",
			listOptions: fleet.ListOptions{OrderKey: "foo", OrderDirection: fleet.OrderAscending},
		},

		// All params defined
		{
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			listOptions: fleet.ListOptions{
				OrderKey:       "foo",
				OrderDirection: fleet.OrderDescending,
				Page:           1,
				PerPage:        100,
			},
		},

		// various error cases
		{
			url:       "/foo?page=foo&per_page=10",
			shouldErr: true,
		},
		{
			url:       "/foo?page=1&per_page=foo",
			shouldErr: true,
		},
		{
			url:       "/foo?page=-1",
			shouldErr: true,
		},
		{
			url:       "/foo?page=-1&per_page=-10",
			shouldErr: true,
		},
		{
			url:       "/foo?page=1&order_direction=desc",
			shouldErr: true,
		},
		{
			url:       "/foo?&order_direction=foo&order_key=",
			shouldErr: true,
		},
	}

	for _, tt := range listOptionsTests {
		t.Run(tt.url, func(t *testing.T) {
			url, _ := url.Parse(tt.url)
			req := &http.Request{URL: url}
			opt, err := listOptionsFromRequest(req)

			if tt.shouldErr {
				assert.NotNil(t, err)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.listOptions, opt)

		})
	}
}
