/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package storage

import (
	"context"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/Uptycs/kubequery/internal/k8s"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type volumeAttachment struct {
	k8s.CommonFields
	v1.VolumeAttachmentSpec
	v1.VolumeAttachmentStatus
}

// VolumeAttachmentColumns returns kubernetes volume attachment fields as Osquery table columns.
func VolumeAttachmentColumns() []table.ColumnDefinition {
	return k8s.GetSchema(&volumeAttachment{})
}

// VolumeAttachmentsGenerate generates the kubernetes volume attachments as Osquery table data.
func VolumeAttachmentsGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	options := metav1.ListOptions{}
	results := make([]map[string]string, 0)

	for {
		vas, err := k8s.GetClient().StorageV1().VolumeAttachments().List(ctx, options)
		if err != nil {
			return nil, err
		}

		for _, va := range vas.Items {
			item := &volumeAttachment{
				CommonFields:           k8s.GetCommonFields(va.ObjectMeta),
				VolumeAttachmentSpec:   va.Spec,
				VolumeAttachmentStatus: va.Status,
			}
			results = append(results, k8s.ToMap(item))
		}

		if vas.Continue == "" {
			break
		}
		options.Continue = vas.Continue
	}

	return results, nil
}
