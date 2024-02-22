/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package apps

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentsGenerate(t *testing.T) {
	ds, err := DeploymentsGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"available_replicas":   "0",
			"cluster_uid":          "blah",
			"creation_timestamp":   "0",
			"host_ipc":             "0",
			"host_network":         "0",
			"host_pid":             "0",
			"min_ready_seconds":    "0",
			"observed_generation":  "0",
			"paused":               "0",
			"ready_replicas":       "0",
			"replicas":             "0",
			"strategy":             "{}",
			"unavailable_replicas": "0",
			"updated_replicas":     "0",
		},
	}, ds)
}

func TestDeploymentContainersGenerate(t *testing.T) {
	ds, err := DeploymentContainersGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{}, ds)
}

func TestDeploymentVolumesGenerate(t *testing.T) {
	ds, err := DeploymentVolumesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{}, ds)
}
