# All API endpoints

```json
{
  "api_endpoints": [
    {
      "display_name": "Log in",
      "protocol": "POST",
      "path": "/api/v1/fleet/login",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Log out",
      "protocol": "POST",
      "path": "/api/v1/fleet/logout",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Forgot password",
      "protocol": "POST",
      "path": "/api/v1/fleet/forgot_password",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update password",
      "protocol": "POST",
      "path": "/api/v1/fleet/change_password",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Reset password",
      "protocol": "POST",
      "path": "/api/v1/fleet/reset_password",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Me",
      "protocol": "GET",
      "path": "/api/v1/fleet/me",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Perform required password reset",
      "protocol": "POST",
      "path": "/api/v1/fleet/perform_required_password_reset",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "SSO config",
      "protocol": "GET",
      "path": "/api/v1/fleet/sso",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Initiate SSO",
      "protocol": "POST",
      "path": "/api/v1/fleet/sso",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "SSO callback",
      "protocol": "POST",
      "path": "/api/v1/fleet/sso/callback",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List activities",
      "protocol": "GET",
      "path": "/api/v1/fleet/activities",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Connect certificate authority (CA)",
      "protocol": "POST",
      "path": "/api/v1/fleet/certificate_authorities",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Add certificate template",
      "protocol": "POST",
      "path": "/api/v1/fleet/certificates",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update certificate authority (CA)",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/certificate_authorities/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List certificate authorities (CAs)",
      "protocol": "GET",
      "path": "/api/v1/fleet/certificate_authorities",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get certificate authority (CA)",
      "protocol": "GET",
      "path": "/api/v1/fleet/certificate_authorities/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List certificate templates",
      "protocol": "GET",
      "path": "/api/v1/fleet/certificates",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get certificate template",
      "protocol": "GET",
      "path": "/api/v1/fleet/certificates/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete certificate authority (CA)",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/certificate_authorities/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete certificate template",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/certificates/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Request certificate",
      "protocol": "POST",
      "path": "/api/v1/fleet/certificate_authorities/:id/request_certificate",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Okta certificate",
      "protocol": "GET",
      "path": "/api/v1/fleet/conditional_access/idp/signing_cert",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Okta configuration profile",
      "protocol": "GET",
      "path": "/api/v1/fleet/conditional_access/idp/apple/profile",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete Microsoft Entra ID",
      "protocol": "DELETE",
      "path": "/api/v1/conditional-access/microsoft",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List carves",
      "protocol": "GET",
      "path": "/api/v1/fleet/carves",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get carve",
      "protocol": "GET",
      "path": "/api/v1/fleet/carves/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get carve block",
      "protocol": "GET",
      "path": "/api/v1/fleet/carves/:id/block/:block_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Fleet certificate",
      "protocol": "GET",
      "path": "/api/v1/fleet/config/certificate",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get configuration",
      "protocol": "GET",
      "path": "/api/v1/fleet/config",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update configuration",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/config",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get global enroll secrets",
      "protocol": "GET",
      "path": "/api/v1/fleet/spec/enroll_secret",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update global enroll secrets",
      "protocol": "POST",
      "path": "/api/v1/fleet/spec/enroll_secret",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get fleet enroll secrets",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets/:id/secrets",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update fleet enroll secrets",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/fleets/:id/secrets",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get version",
      "protocol": "GET",
      "path": "/api/v1/fleet/version",
      "deprecated": false,
      "roles": []
    },
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
      "display_name": "Add label",
      "protocol": "POST",
      "path": "/api/v1/fleet/labels",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update label",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/labels/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get label",
      "protocol": "GET",
      "path": "/api/v1/fleet/labels/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get labels summary",
      "protocol": "GET",
      "path": "/api/v1/fleet/labels/summary",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List labels",
      "protocol": "GET",
      "path": "/api/v1/fleet/labels",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List label's hosts",
      "protocol": "GET",
      "path": "/api/v1/fleet/labels/:id/hosts",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete label by name",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/labels/:name",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete label by ID",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/labels/id/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create custom OS setting (configuration profile)",
      "protocol": "POST",
      "path": "/api/v1/fleet/configuration_profiles",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List custom OS settings (configuration profiles)",
      "protocol": "GET",
      "path": "/api/v1/fleet/configuration_profiles",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get or download custom OS setting (configuration profile)",
      "protocol": "GET",
      "path": "/api/v1/fleet/configuration_profiles/:profile_uuid",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete custom OS setting (configuration profile)",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/configuration_profiles/:profile_uuid",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Resend custom OS setting (configuration profile)",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/configuration_profiles/:profile_uuid/resend",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Batch-update custom OS settings (configuration profiles)",
      "protocol": "POST",
      "path": "/api/v1/fleet/configuration_profiles/batch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Resend custom OS setting (configuration profile) by Fleet Desktop token",
      "protocol": "POST",
      "path": "/api/v1/fleet/device/:token/configuration_profiles/:profile_uuid/resend",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Batch-resend custom OS setting (configuration profile)",
      "protocol": "POST",
      "path": "/api/v1/fleet/configuration_profiles/resend/batch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update disk encryption",
      "protocol": "POST",
      "path": "/api/v1/fleet/disk_encryption",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get disk encryption status",
      "protocol": "GET",
      "path": "/api/v1/fleet/disk_encryption",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get OS settings (configuration profiles) status",
      "protocol": "GET",
      "path": "/api/v1/fleet/configuration_profiles/summary",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get OS setting (configuration profile) status",
      "protocol": "GET",
      "path": "/api/v1/fleet/configuration_profile/:profile_uuid/status",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update custom MDM setup enrollment profile",
      "protocol": "POST",
      "path": "/api/v1/fleet/enrollment_profiles/automatic",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get custom MDM setup enrollment profile",
      "protocol": "GET",
      "path": "/api/v1/fleet/enrollment_profiles/automatic",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete custom MDM setup enrollment profile",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/enrollment_profiles/automatic",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Over-the-Air (OTA) enrollment profile",
      "protocol": "GET",
      "path": "/api/v1/fleet/enrollment_profiles/ota",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get manual enrollment profile",
      "protocol": "GET",
      "path": "/api/v1/fleet/enrollment_profiles/manual",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create bootstrap package",
      "protocol": "POST",
      "path": "/api/v1/fleet/bootstrap",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get bootstrap package metadata",
      "protocol": "GET",
      "path": "/api/v1/fleet/bootstrap/:fleet_id/metadata",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete bootstrap package",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/bootstrap/:fleet_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Download bootstrap package",
      "protocol": "GET",
      "path": "/api/v1/fleet/bootstrap",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get bootstrap package status",
      "protocol": "GET",
      "path": "/api/v1/fleet/bootstrap/summary",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update setup experience",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/setup_experience",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create EULA",
      "protocol": "POST",
      "path": "/api/v1/fleet/setup_experience/eula",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get EULA metadata",
      "protocol": "GET",
      "path": "/api/v1/fleet/setup_experience/eula/metadata",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete EULA",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/setup_experience/eula/:token",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Download EULA",
      "protocol": "GET",
      "path": "/api/v1/fleet/setup_experience/eula/:token",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List setup experience software",
      "protocol": "GET",
      "path": "/api/v1/fleet/setup_experience/software",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update setup experience software",
      "protocol": "PUT",
      "path": "/api/v1/fleet/setup_experience/software",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create setup experience script",
      "protocol": "POST",
      "path": "/api/v1/fleet/setup_experience/script",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update setup experience script",
      "protocol": "PUT",
      "path": "/api/v1/fleet/setup_experience/script",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get or download setup experience script",
      "protocol": "GET",
      "path": "/api/v1/fleet/setup_experience/script",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete setup experience script",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/setup_experience/script",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Run MDM command",
      "protocol": "POST",
      "path": "/api/v1/fleet/commands/run",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get MDM command results",
      "protocol": "GET",
      "path": "/api/v1/fleet/commands/results",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List MDM commands",
      "protocol": "GET",
      "path": "/api/v1/fleet/commands",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Apple Push Notification service (APNs)",
      "protocol": "GET",
      "path": "/api/v1/fleet/apns",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List Apple Business Manager (ABM) tokens",
      "protocol": "GET",
      "path": "/api/v1/fleet/abm_tokens",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List Volume Purchasing Program (VPP) tokens",
      "protocol": "GET",
      "path": "/api/v1/fleet/vpp_tokens",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get identity provider (IdP) details",
      "protocol": "GET",
      "path": "/api/v1/fleet/scim/details",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Android Enterprise",
      "protocol": "GET",
      "path": "/api/v1/fleet/android_enterprise",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List policies",
      "protocol": "GET",
      "path": "/api/v1/fleet/global/policies",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List fleet policies",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets/:id/policies",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get policies count",
      "protocol": "GET",
      "path": "/api/v1/fleet/policies/count",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get fleet policies count",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets/:fleet_id/policies/count",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get policy",
      "protocol": "GET",
      "path": "/api/v1/fleet/global/policies/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get fleet policy",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create policy",
      "protocol": "POST",
      "path": "/api/v1/fleet/global/policies",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create fleet policy",
      "protocol": "POST",
      "path": "/api/v1/fleet/fleets/:id/policies",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete policies",
      "protocol": "POST",
      "path": "/api/v1/fleet/global/policies/delete",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete fleet policies",
      "protocol": "POST",
      "path": "/api/v1/fleet/fleets/:fleet_id/policies/delete",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update policy",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/global/policies/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update fleet policy",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/fleets/:fleet_id/policies/:policy_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Reset policy automations",
      "protocol": "POST",
      "path": "/api/v1/fleet/automations/reset",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List reports",
      "protocol": "GET",
      "path": "/api/v1/fleet/reports",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get report",
      "protocol": "GET",
      "path": "/api/v1/fleet/reports/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get report data",
      "protocol": "GET",
      "path": "/api/v1/fleet/report/:id/report",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get host's report data",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/reports/:report_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create report",
      "protocol": "POST",
      "path": "/api/v1/fleet/reports",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update report",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/reports/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete report by name",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/reports/:name",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete report by ID",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/reports/id/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete reports",
      "protocol": "POST",
      "path": "/api/v1/fleet/reports/delete",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Run live report",
      "protocol": "POST",
      "path": "/api/v1/fleet/reports/:id/run",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Run script",
      "protocol": "POST",
      "path": "/api/v1/fleet/scripts/run",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get script result",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts/results/:execution_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Batch-run script",
      "protocol": "POST",
      "path": "/api/v1/fleet/scripts/run/batch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List batch scripts",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts/batch",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get batch script",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts/batch/:batch_execution_id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List hosts targeted in batch script",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts/batch/:batch_execution_id/host_results",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Cancel batch script",
      "protocol": "POST",
      "path": "/scripts/batch/abc-def/cancel",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create script",
      "protocol": "POST",
      "path": "/api/v1/fleet/scripts",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update script",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/scripts/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete script",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/scripts/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List scripts",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List host's scripts",
      "protocol": "GET",
      "path": "/api/v1/fleet/hosts/:id/scripts",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get or download script",
      "protocol": "GET",
      "path": "/api/v1/fleet/scripts/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get session",
      "protocol": "GET",
      "path": "/api/v1/fleet/sessions/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete session",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/sessions/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List software",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/titles",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List software versions",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/versions",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List operating systems",
      "protocol": "GET",
      "path": "/api/v1/fleet/os_versions",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get software",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/titles/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get software version",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/versions/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get operating system version",
      "protocol": "GET",
      "path": "/api/v1/fleet/os_versions/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Add package",
      "protocol": "POST",
      "path": "/api/v1/fleet/software/package",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update package",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/software/titles/:id/package",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update software icon",
      "protocol": "PUT",
      "path": "/api/v1/fleet/software/titles/:id/icon",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Download software icon",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/titles/:id/icon",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete software icon",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/software/titles/:id/icon",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List Apple App Store apps",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/app_store_apps",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Add app store app",
      "protocol": "POST",
      "path": "/api/v1/fleet/software/app_store_apps",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update app store app",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/software/titles/:title_id/app_store_app",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List Fleet-maintained apps",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/fleet_maintained_apps",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get Fleet-maintained app",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/fleet_maintained_apps/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create Fleet-maintained app",
      "protocol": "POST",
      "path": "/api/v1/fleet/software/fleet_maintained_apps",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Download software",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/titles/:id/package?alt=media",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Install software",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/software/:software_title_id/install",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Uninstall software",
      "protocol": "POST",
      "path": "/api/v1/fleet/hosts/:id/software/:software_title_id/uninstall",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get software install result",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/install/:install_uuid/results",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Download package",
      "protocol": "GET",
      "path": "/api/v1/fleet/software/titles/:software_title_id/package?alt=media",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete software",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/software/titles/:software_title_id/available_for_install",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List vulnerabilities",
      "protocol": "GET",
      "path": "/api/v1/fleet/vulnerabilities",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get vulnerability",
      "protocol": "GET",
      "path": "/api/v1/fleet/vulnerabilities/:cve",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Search targets",
      "protocol": "POST",
      "path": "/api/v1/fleet/targets",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List fleets",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get fleet",
      "protocol": "GET",
      "path": "/api/v1/fleet/fleets/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create fleet",
      "protocol": "POST",
      "path": "/api/v1/fleet/fleets",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update fleet",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/fleets/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Add users to fleet",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/fleets/:id/users",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update fleet's agent options",
      "protocol": "POST",
      "path": "/api/v1/fleet/fleets/:id/agent_options",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete fleet",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/fleets/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Translate IDs",
      "protocol": "POST",
      "path": "/api/v1/fleet/translate",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List users",
      "protocol": "GET",
      "path": "/api/v1/fleet/users",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create user",
      "protocol": "POST",
      "path": "/api/v1/fleet/users/admin",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create user from invite",
      "protocol": "POST",
      "path": "/api/v1/fleet/users",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get user",
      "protocol": "GET",
      "path": "/api/v1/fleet/users/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update user",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/users/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete user",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/users/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Require password reset",
      "protocol": "POST",
      "path": "/api/v1/fleet/users/:id/require_password_reset",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List sessions",
      "protocol": "GET",
      "path": "/api/v1/fleet/users/:id/sessions",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete sessions",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/users/:id/sessions",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Invite user",
      "protocol": "POST",
      "path": "/api/v1/fleet/invites",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List invites",
      "protocol": "GET",
      "path": "/api/v1/fleet/invites",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete invite",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/invites/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Verify invite",
      "protocol": "GET",
      "path": "/api/v1/fleet/invites/:token",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Update invite",
      "protocol": "PATCH",
      "path": "/api/v1/fleet/invites/:id",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get errors",
      "protocol": "GET",
      "path": "/debug/errors",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get database information",
      "protocol": "GET",
      "path": "/debug/db/:key",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Get profiling information",
      "protocol": "GET",
      "path": "/debug/pprof/:key",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "List custom variables",
      "protocol": "GET",
      "path": "/api/v1/fleet/custom_variables",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Create custom variable",
      "protocol": "POST",
      "path": "/api/v1/fleet/custom_variables",
      "deprecated": false,
      "roles": []
    },
    {
      "display_name": "Delete custom variable",
      "protocol": "DELETE",
      "path": "/api/v1/fleet/custom_variables/:id",
      "deprecated": false,
      "roles": []
    }
  ]
}
```
