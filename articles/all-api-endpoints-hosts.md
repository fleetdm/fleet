# Hosts API endpoints

```json
{
  "api_endpoints": [
    {
      "display_name": "List hosts",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Count hosts",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/count",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get hosts summary",
      "protocol": "GET",
      "path": "/api/v1/fleet/host_summary",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host by identifier",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/identifier/:identifier",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host by Fleet Desktop token",
      "protocol": "GET",
      "path": "/api/v1/fleet/device/:token",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete host",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/hosts/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Refetch host",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/refetch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Refetch host by Fleet Desktop token",
      "protocol": "POST",
      "path": "/api/v1/fleet/device/:token/refetch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update hosts' fleet",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/transfer",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update hosts' fleet by filter",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/transfer/filter",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Turn off host's MDM",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/hosts/:id/mdm",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Batch-delete hosts",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/delete",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update human-device mapping",
      "protocol": "PUT",
      "path": "/api/v1/fleet/hosts/:id/device_mapping",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's device health report",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/health",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's mobile device management (MDM) information",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/mdm",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get mobile device management (MDM) status",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/summary/mdm",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's mobile device management (MDM) and Munki information",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/macadmins",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get hosts' aggregate mobile device management (MDM) and Munki information",
      "protocol": "GET",
      "path": "/api/v1/fleet/macadmins",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's software",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/software",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get hosts report in CSV",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/report",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's disk encryption key",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/encryption_key",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's certificates",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/certificates",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's OS settings (configuration profile)",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/configuration_profiles",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Lock host",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/lock",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Unlock host",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/unlock",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Wipe host",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/wipe",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's past activity",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/activities",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's upcoming activity",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/activities/upcoming",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Cancel host's upcoming activity",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/hosts/:id/activities/upcoming/:activity_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Add labels to host",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/labels",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Remove labels from host",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/hosts/:id/labels",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Run live query on host (ad hoc)",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/query",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Run live query on host by identifier (ad hoc)",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/identifier/:identifier/query",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Bypass host's conditional access",
      "protocol": "POST",
      "path": "/api/v1/fleet/device/:token/bypass_conditional_access",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get human-device mapping",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/device_mapping",
      "deprecated": true,
      "roles": []
    }
  ]
}
```
