/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package main

import (
	"context"
	"fmt"

	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	err := k8s.Init()
	if err != nil {
		panic(fmt.Sprintf("Error connecting to kubernetes API server: %s", err))
	}

	options := v1.GetOptions{}
	ks, err := k8s.GetClient().CoreV1().Namespaces().Get(context.Background(), "kube-system", options)
	if err != nil {
		panic(fmt.Sprintf("Error getting kube-system namespace: %s", err))
	}

	fmt.Print(ks.ObjectMeta.UID)
}
