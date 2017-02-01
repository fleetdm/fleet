package service

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestListOptionsFromRequest(t *testing.T) {
	var listOptionsTests = []struct {
		// url string to parse
		url string
		// expected list options
		listOptions kolide.ListOptions
		// should cause an error
		shouldErr bool
	}{
		// both params provided
		{
			url:         "/foo?page=1&per_page=10",
			listOptions: kolide.ListOptions{Page: 1, PerPage: 10},
		},
		// only per_page (page should default to 0)
		{
			url:         "/foo?per_page=10",
			listOptions: kolide.ListOptions{Page: 0, PerPage: 10},
		},
		// only page (per_page should default to defaultPerPage
		{
			url:         "/foo?page=10",
			listOptions: kolide.ListOptions{Page: 10, PerPage: defaultPerPage},
		},
		// no params provided (defaults to empty ListOptions indicating
		// unlimited)
		{
			url:         "/foo?unrelated=foo",
			listOptions: kolide.ListOptions{},
		},

		// Both order params provided
		{
			url:         "/foo?order_key=foo&order_direction=desc",
			listOptions: kolide.ListOptions{OrderKey: "foo", OrderDirection: kolide.OrderDescending},
		},
		// Both order params provided (asc)
		{
			url:         "/foo?order_key=bar&order_direction=asc",
			listOptions: kolide.ListOptions{OrderKey: "bar", OrderDirection: kolide.OrderAscending},
		},
		// Default order direction
		{
			url:         "/foo?order_key=foo",
			listOptions: kolide.ListOptions{OrderKey: "foo", OrderDirection: kolide.OrderAscending},
		},

		// All params defined
		{
			url: "/foo?order_key=foo&order_direction=desc&page=1&per_page=100",
			listOptions: kolide.ListOptions{
				OrderKey:       "foo",
				OrderDirection: kolide.OrderDescending,
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
