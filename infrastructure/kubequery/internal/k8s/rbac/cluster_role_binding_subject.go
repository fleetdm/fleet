/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package rbac

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterRoleBindingSubject struct {
	k8s.CommonFields
	RoleAPIGroup     string
	RoleName         string
	RoleKind         string
	SubjectName      string
	SubjectKind      string
	SubjectNamespace string
}

// ClusterRoleBindingSubjectColumns returns kubernetes cluster role binding subject fields as Osquery table columns.
func ClusterRoleBindingSubjectColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&clusterRoleBindingSubject{})
}

// ClusterRoleBindingSubjectsGenerate generates the kubernetes cluster role binding subjects as Osquery table data.
func ClusterRoleBindingSubjectsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		crbs, err := k8s.GetClient().RbacV1().ClusterRoleBindings().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, crb := range crbs.Items {
			for _, s := range crb.Subjects {
				item := &clusterRoleBindingSubject{
					CommonFields:     k8s.GetCommonFields(crb.ObjectMeta),
					RoleAPIGroup:     crb.RoleRef.APIGroup,
					RoleName:         crb.RoleRef.Name,
					RoleKind:         crb.RoleRef.Kind,
					SubjectName:      s.Name,
					SubjectKind:      s.Kind,
					SubjectNamespace: s.Namespace,
				}
				results = append(results, k8s.ToMap(item))
			}
		}

		if crbs.Continue == "" {
			break
		}
		options.Continue = crbs.Continue
	}

	return results, nil
}
