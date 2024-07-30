/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package policy

import (
	"context"
	"testing"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/stretchr/testify/assert"
)

func TestPodSecurityPoliciesGenerate(t *testing.T) {
	psps, err := PodSecurityPoliciesGenerate(context.TODO(), table.QueryContext{})
	assert.Nil(t, err)
	assert.Equal(t, []map[string]string{
		{
			"allow_privilege_escalation": "0",
			"annotations":                "{\"apparmor.security.beta.kubernetes.io/allowedProfileNames\":\"runtime/default\",\"apparmor.security.beta.kubernetes.io/defaultProfileName\":\"runtime/default\",\"kubectl.kubernetes.io/last-applied-configuration\":\"{\\\"apiVersion\\\":\\\"policy/v1beta1\\\",\\\"kind\\\":\\\"PodSecurityPolicy\\\",\\\"metadata\\\":{\\\"annotations\\\":{\\\"apparmor.security.beta.kubernetes.io/allowedProfileNames\\\":\\\"runtime/default\\\",\\\"apparmor.security.beta.kubernetes.io/defaultProfileName\\\":\\\"runtime/default\\\",\\\"seccomp.security.alpha.kubernetes.io/allowedProfileNames\\\":\\\"docker/default,runtime/default\\\",\\\"seccomp.security.alpha.kubernetes.io/defaultProfileName\\\":\\\"runtime/default\\\"},\\\"name\\\":\\\"restricted\\\"},\\\"spec\\\":{\\\"allowPrivilegeEscalation\\\":false,\\\"fsGroup\\\":{\\\"ranges\\\":[{\\\"max\\\":65535,\\\"min\\\":1}],\\\"rule\\\":\\\"MustRunAs\\\"},\\\"hostIPC\\\":false,\\\"hostNetwork\\\":false,\\\"hostPID\\\":false,\\\"privileged\\\":false,\\\"readOnlyRootFilesystem\\\":false,\\\"requiredDropCapabilities\\\":[\\\"ALL\\\"],\\\"runAsUser\\\":{\\\"rule\\\":\\\"MustRunAsNonRoot\\\"},\\\"seLinux\\\":{\\\"rule\\\":\\\"RunAsAny\\\"},\\\"supplementalGroups\\\":{\\\"ranges\\\":[{\\\"max\\\":65535,\\\"min\\\":1}],\\\"rule\\\":\\\"MustRunAs\\\"},\\\"volumes\\\":[\\\"configMap\\\",\\\"emptyDir\\\",\\\"projected\\\",\\\"secret\\\",\\\"downwardAPI\\\",\\\"persistentVolumeClaim\\\"]}}\\n\",\"seccomp.security.alpha.kubernetes.io/allowedProfileNames\":\"docker/default,runtime/default\",\"seccomp.security.alpha.kubernetes.io/defaultProfileName\":\"runtime/default\"}",
			"cluster_uid":                "b7fd8e77-93de-4742-9037-5db9a01e966a",
			"creation_timestamp":         "1611164232",
			"fs_group":                   "{\"rule\":\"MustRunAs\",\"ranges\":[{\"min\":1,\"max\":65535}]}",
			"host_ipc":                   "0",
			"host_network":               "0",
			"host_pid":                   "0",
			"name":                       "restricted",
			"privileged":                 "0",
			"read_only_root_filesystem":  "0",
			"required_drop_capabilities": "[\"ALL\"]",
			"run_as_user":                "{\"rule\":\"MustRunAsNonRoot\"}",
			"se_linux":                   "{\"rule\":\"RunAsAny\"}",
			"supplemental_groups":        "{\"rule\":\"MustRunAs\",\"ranges\":[{\"min\":1,\"max\":65535}]}",
			"uid":                        "de6eb036-24db-4490-8811-590a2c2e1529",
			"volumes":                    "[\"configMap\",\"emptyDir\",\"projected\",\"secret\",\"downwardAPI\",\"persistentVolumeClaim\"]",
		},
	}, psps)
}
