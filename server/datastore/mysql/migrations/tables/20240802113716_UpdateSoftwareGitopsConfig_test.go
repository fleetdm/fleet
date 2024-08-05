package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20240802113716(t *testing.T) {
	db := applyUpToPrev(t)

	badCfg := `
{
  "mdm": {
    "ios_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_setup": {
      "bootstrap_package": "",
      "macos_setup_assistant": "",
      "enable_end_user_authentication": false,
      "enable_release_device_manually": false
    },
    "macos_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "ipados_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_settings": {
      "custom_settings": []
    },
    "windows_updates": {
      "deadline_days": null,
      "grace_period_days": null
    },
    "windows_settings": {
      "custom_settings": []
    },
    "enable_disk_encryption": false
  },
  "scripts": [],
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true
  },
  "software": [
      {
        "url": "http://localhost:8100/1Password.pkg",
        "self_service": true,
        "install_script": {
          "path": ""
        },
        "pre_install_query": {
          "path": ""
        },
        "post_install_script": {
          "path": ""
        }
      }
    ],
  "integrations": {
    "jira": null,
    "zendesk": null,
    "google_calendar": {
      "webhook_url": "",
      "enable_calendar_events": false
    }
  },
  "webhook_settings": {
    "host_status_webhook": {
      "days_count": 0,
      "destination_url": "",
      "host_percentage": 0,
      "enable_host_status_webhook": false
    },
    "failing_policies_webhook": {
      "policy_ids": null,
      "destination_url": "",
      "host_batch_size": 0,
      "enable_failing_policies_webhook": false
    }
  },
  "host_expiry_settings": {
    "host_expiry_window": 30,
    "host_expiry_enabled": true
  }
}

`

	badCfgEmptyArr := `
{
  "mdm": {
    "ios_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_setup": {
      "bootstrap_package": "",
      "macos_setup_assistant": "",
      "enable_end_user_authentication": false,
      "enable_release_device_manually": false
    },
    "macos_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "ipados_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_settings": {
      "custom_settings": []
    },
    "windows_updates": {
      "deadline_days": null,
      "grace_period_days": null
    },
    "windows_settings": {
      "custom_settings": []
    },
    "enable_disk_encryption": false
  },
  "scripts": [],
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true
  },
  "software": [],
  "integrations": {
    "jira": null,
    "zendesk": null,
    "google_calendar": {
      "webhook_url": "",
      "enable_calendar_events": false
    }
  },
  "webhook_settings": {
    "host_status_webhook": {
      "days_count": 0,
      "destination_url": "",
      "host_percentage": 0,
      "enable_host_status_webhook": false
    },
    "failing_policies_webhook": {
      "policy_ids": null,
      "destination_url": "",
      "host_batch_size": 0,
      "enable_failing_policies_webhook": false
    }
  },
  "host_expiry_settings": {
    "host_expiry_window": 30,
    "host_expiry_enabled": true
  }
}
`

	badCfgNoSoftwareField := `
{
  "mdm": {
    "ios_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_setup": {
      "bootstrap_package": "",
      "macos_setup_assistant": "",
      "enable_end_user_authentication": false,
      "enable_release_device_manually": false
    },
    "macos_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "ipados_updates": {
      "deadline": "",
      "minimum_version": ""
    },
    "macos_settings": {
      "custom_settings": []
    },
    "windows_updates": {
      "deadline_days": null,
      "grace_period_days": null
    },
    "windows_settings": {
      "custom_settings": []
    },
    "enable_disk_encryption": false
  },
  "scripts": [],
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true
  },
  "integrations": {
    "jira": null,
    "zendesk": null,
    "google_calendar": {
      "webhook_url": "",
      "enable_calendar_events": false
    }
  },
  "webhook_settings": {
    "host_status_webhook": {
      "days_count": 0,
      "destination_url": "",
      "host_percentage": 0,
      "enable_host_status_webhook": false
    },
    "failing_policies_webhook": {
      "policy_ids": null,
      "destination_url": "",
      "host_batch_size": 0,
      "enable_failing_policies_webhook": false
    }
  },
  "host_expiry_settings": {
    "host_expiry_window": 30,
    "host_expiry_enabled": true
  }
}
`

	tid1 := execNoErrLastID(t, db, `INSERT INTO teams (name, config) VALUES (?,?)`, "team 1", badCfg)
	tid2 := execNoErrLastID(t, db, `INSERT INTO teams (name, config) VALUES (?,?)`, "team 2", badCfgEmptyArr)
	tid3 := execNoErrLastID(t, db, `INSERT INTO teams (name, config) VALUES (?,?)`, "team 3", badCfgNoSoftwareField)

	// Apply current migration.
	applyNext(t, db)

	var team fleet.Team
	require.NoError(t, db.Get(&team, "SELECT id, config FROM teams WHERE id = ?", tid1))

	// Team with a package should see it in the new field
	require.NotNil(t, team.Config.Software)
	require.True(t, team.Config.Software.Packages.Set)
	require.True(t, team.Config.Software.Packages.Valid)
	require.Len(t, team.Config.Software.Packages.Value, 1)

	require.False(t, team.Config.Software.AppStoreApps.Set)
	require.False(t, team.Config.Software.AppStoreApps.Valid)
	require.Len(t, team.Config.Software.AppStoreApps.Value, 0)

	team = fleet.Team{}
	require.NoError(t, db.Get(&team, "SELECT id, config FROM teams WHERE id = ?", tid2))

	// Team with an empty array originally should have JSON null set for packages
	require.NotNil(t, team.Config.Software)
	require.True(t, team.Config.Software.Packages.Set)
	require.True(t, team.Config.Software.Packages.Valid)
	require.Len(t, team.Config.Software.Packages.Value, 0)

	require.False(t, team.Config.Software.AppStoreApps.Set)
	require.False(t, team.Config.Software.AppStoreApps.Valid)
	require.Len(t, team.Config.Software.AppStoreApps.Value, 0)

	team = fleet.Team{}
	require.NoError(t, db.Get(&team, "SELECT id, config FROM teams WHERE id = ?", tid3))

	require.Nil(t, team.Config.Software)
}
