/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package discovery

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInfoGenerate(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		Major:      "1",
		Minor:      "46",
		GitVersion: "master",
		GitCommit:  "123",
		BuildDate:  "1970-01-01T00:00:00Z",
		GoVersion:  "go1.15",
		Compiler:   "gc",
		Platform:   "linux/amd64",
	}
	k8s.SetClient(clientset, types.UID("hello"), "cluster-name")

	ars, err := InfoGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"build_date":   "1970-01-01T00:00:00Z",
			"cluster_uid":  "hello",
			"cluster_name": "cluster-name",
			"compiler":     "gc",
			"git_commit":   "123",
			"git_version":  "master",
			"go_version":   "go1.15",
			"major":        "1",
			"minor":        "46",
			"platform":     "linux/amd64",
		},
	}, ars)
}
