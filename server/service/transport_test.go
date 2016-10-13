package service

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
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
