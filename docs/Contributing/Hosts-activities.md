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

## Past activities

Examples for each past activity type.



