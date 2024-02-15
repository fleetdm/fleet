# Hosts activities

This document includes the API responses for each host's upcopming and past activity type.

- [Upcoming activities](#upcoming-activities)
- [Past activities](#past-activities)

## Upcoming activities

Examples for each upcoming activity type.

### `ran_script`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "46716e6b-2859-4ca6-9c80-becfc8f38e12",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "ran_script",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "script",
    "script_name": "set-timezone.sh",
    "script_execution_id": "bc1ede69-7b78-4137-a20e-3469e7f7eeb9",
    "exit_code": null,     
    "async": true,   
  }
}
```

### `ran_mdm_command`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "db86f163-6712-460b-86ed-c95696a14df2",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "ran_mdm_command",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "db84026b-1c53-4685-bb9c-56cd274f6e5b",
    "status": "Pending",
  }
}
```

### `installed_fleetd`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "823A8ECE-F974-4423-ABED-0508630D041B",
  "actor_full_name": "Fleet",
  "actor_gravatar": "",
  "actor_email": "",
  "type": "installed_fleetd",
  "fleet_initiated_activity": true,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "61E56080-1FFE-42A5-BD51-43D4174CB47F",
    "status": "Pending",
  }
}
```

### `set_account_configuration`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "db86f163-6712-460b-86ed-c95696a14df2",
  "actor_full_name": "Fleet",
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "set_account_configuration",
  "fleet_initiated_activity": true,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "db84026b-1c53-4685-bb9c-56cd274f6e5b",
    "status": "Pending",
  }
}
```

### `created_macos_profile`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "actor_full_name": "Marko",
  "uuid": "4532746b-3052-414a-b35b-ed6fd458ac30",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "created_macos_profile",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "profile_name": "macOS restrictions",
    "type": "mdm_command",
    "command_uuid": "eeeddb94-52d3-4071-8b18-7322cd382abb",
    "status": "Pending",
  }
}
```

### `edited_macos_profile`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "actor_full_name": "GitOps user",
  "uuid": "88e8622e-0f1c-497b-923b-623ce6b5eb88",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "",
  "type": "edited_macos_profile",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "profile_name": "Restrictions",
    "type": "mdm_command",
    "command_uuid": "b2c0b113-ca79-41ee-87d5-724de51b352c",
    "status": "Pending",
  }
}
```

### `deleted_macos_profile`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "4532746b-3052-414a-b35b-ed6fd458ac30",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "deleted_macos_profile",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "profile_name": "macOS restrictions",
    "type": "mdm_command",
    "command_uuid": "68d81b61-78a0-48f6-926a-9c441e7ff961",
    "status": "Pending",
  }
}
```

### `enabled_disk_encryption`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "ad9cdf4e-7167-40c4-a4d8-601110764049",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "enabled_disk_encryption",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "f4ebbb4e-b2cf-4b60-92c8-df566de2e64d",
    "status": "Pending",
  }
}
```

### `disabled_disk_encryption`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "2ff4af60-e226-4550-98e0-36cdfcd80706",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "disabled_disk_encryption",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "f4ebbb4e-b2cf-4b60-92c8-df566de2e64d",
    "status": "Pending",
  }
},
```

### `edited​_macos​_min​_version`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "2ff4af60-e226-4550-98e0-36cdfcd80706",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "edited​_macos​_min​_version",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "minimum_version": "14.3.1",
    "deadline": "2023-12-31",   
    "status": "Pending",    
  }
}
```

### `locked`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "ef4c5b68-a297-4abe-adb0-818589800513",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "locked",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "de05f360-0c37-4665-bfae-f5c9c48d9d50",   
    "status": "Pending",   
  }
}
```

### `wiped`

```json
{
  "created_at": "2023-07-27T14:35:08Z",
  "uuid": "e6575132-0366-47a6-bc37-24e433581752",
  "actor_full_name": "Marko",
  "actor_id": 1,
  "actor_gravatar": "",
  "actor_email": "marko@example.com",
  "type": "wiped",
  "fleet_initiated_activity": false,
  "details": {
    "host_id": 1,
    "host_display_name": "Steve's MacBook Pro",
    "type": "mdm_command",
    "command_uuid": "f43efadb-13b3-4d63-8a68-f6de4d3277e4",
    "status": "Pending",        
  }
}
```

## Past activities

Examples for each past activity type.

### `ran_script`

```json

```

