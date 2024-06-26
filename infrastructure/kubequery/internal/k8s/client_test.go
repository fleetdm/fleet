/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetClient(t *testing.T) {
	SetClient(fake.NewSimpleClientset(), types.UID("uid"), "cluster-name")
	assert.NotNil(t, GetClient(), "Clientset should be valid")
}

func TestGetClusterUID(t *testing.T) {
	SetClient(fake.NewSimpleClientset(), types.UID("uid"), "cluster-name")
	assert.Equal(t, types.UID("uid"), GetClusterUID())
}

func TestGetClusterName(t *testing.T) {
	SetClient(fake.NewSimpleClientset(), types.UID("uid"), "cluster-name")
	assert.Equal(t, "cluster-name", GetClusterName())
}
