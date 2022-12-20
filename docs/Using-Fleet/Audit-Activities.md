<!-- DO NOT EDIT. This document is automatically generated. -->
# Audit Activities

Fleet logs the following information for administrative actions (in JSON):

- `created_at`: Timestamp of the event.
- `id`: Unique ID of the generated event in Fleet.
- `actor_full_name`: Author user name (missing if the user was deleted).
- `actor_id`: Unique ID of the author in Fleet (missing if the user was deleted).
- `actor_gravatar`: Gravatar URL of the author (missing if the user was deleted).
- `actor_email`: E-mail of the author (missing if the user was deleted).
- `type`: Type of the activity (see all types below).
- `details`: Specific details depending on the type of activity (see details for each activity type below).

Example:
```json
{
	"created_at": "2022-12-20T14:54:17Z",
	"id": 6,
	"actor_full_name": "Gandalf",
	"actor_id": 2,
	"actor_gravatar": "foo@example.com",
	"actor_email": "foo@example.com",
	"type": "edited_saved_query",
	"details":{
		"query_id": 42,
		"query_name": "Some query name"
	}
}
```
	
## List of activities and their specific details

### Type `created_pack`

Generated when creating scheduled query packs.

This activity contains the following fields:
- "pack_id": the id of the created pack.
- "pack_name": the name of the created pack.

#### Example

```json
{
	"pack_id": 123, 
	"pack_name": "foo"
}
```

### Type `edited_pack`

Generated when editing scheduled query packs.

This activity contains the following fields:
- "pack_id": the id of the edited pack.
- "pack_name": the name of the edited pack.

#### Example

```json
{
	"pack_id": 123, 
	"pack_name": "foo"
}
```

### Type `deleted_pack`

Generated when deleting scheduled query packs.

This activity contains the following fields:
- "pack_name": the name of the created pack.

#### Example

```json
{
	"pack_name": "foo"
}
```

### Type `applied_spec_pack`

Generated when applying a scheduled query pack spec.

This activity does not contain any detail fields.

### Type `created_policy`

Generated when creating policies.

This activity contains the following fields:
- "policy_id": the ID of the created policy.
- "policy_name": the name of the created policy.

#### Example

```json
{
	"policy_id": 123, 
	"policy_name": "foo"
}
```

### Type `edited_policy`

Generated when editing policies.

This activity contains the following fields:
- "policy_id": the ID of the edited policy.
- "policy_name": the name of the edited policy.

#### Example

```json
{
	"policy_id": 123, 
	"policy_name": "foo"
}
```

### Type `deleted_policy`

Generated when deleting policies.

This activity contains the following fields:
- "policy_id": the ID of the deleted policy.
- "policy_name": the name of the deleted policy.

#### Example

```json
{
	"policy_id": 123, 
	"policy_name": "foo"
}
```

### Type `applied_spec_policy`

Generated when applying policy specs.

This activity contains a field "policies" where each item is a policy spec with the following fields:
- "name": Name of the applied policy.
- "query": SQL query of the policy
- "description": Description of the policy
- "critical": Marks the policy as high impact
- "resolution": Describes how to solve a failing policy.
- "team": Name of the team this policy belongs to.
- "platform": Comma-separated string to indicate the target platforms.


#### Example

```json
{
	"policies": [
		{
			"name":"Gatekeeper enabled (macOS)",
			"query":"SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
			"critical":false,
			"platform":"darwin",
			"resolution":"To enable Gatekeeper, on the failing device [...]",
			"description":"Checks to make sure that the Gatekeeper feature is [...]"
		},
		{
			"name":"Full disk encryption enabled (Windows)",
			"query":"SELECT 1 FROM bitlocker_info WHERE drive_letter='C:' AND protection_status=1;",
			"critical":false,
			"platform":"windows",
			"resolution":"To get additional information, run the following osquery [...]",
			"description":"Checks to make sure that full disk encryption is enabled on Windows devices."
		}
	]
}
```

### Type `created_saved_query`

Generated when creating a new query.

This activity contains the following fields:
- "query_id": the ID of the created query.
- "query_name": the name of the created query.

#### Example

```json
{
	"query_id": 123, 
	"query_name": "foo"
}
```

### Type `edited_saved_query`

Generated when editing a saved query.

This activity contains the following fields:
- "query_id": the ID of the query being edited.
- "query_name": the name of the query being edited.

#### Example

```json
{
	"query_id": 123, 
	"query_name": "foo"
}
```

### Type `deleted_saved_query`

Generated when deleting a saved query.

This activity contains the following fields:
- "query_name": the name of the query being deleted.

#### Example

```json
{
	"query_name": "foo"
}
```

### Type `deleted_multiple_saved_query`

Generated when deleting multiple saved queries.

This activity contains the following fields:
- "query_ids": list of IDs of the deleted saved queries.

#### Example

```json
{
	"query_ids": [1, 42, 100]
}
```

### Type `applied_spec_saved_query`

Generated when applying a query spec.

This activity contains a field "specs" where each item is a query spec with the following fields:
- "name": Name of the query.
- "description": Description of the query.
- "query": SQL query.

#### Example

```json
{
	"specs": [
		{
			"name":"Get OpenSSL versions",
			"query":"SELECT name AS name, version AS version, 'deb_packages' AS source FROM [...]",
			"description":"Retrieves the OpenSSL version."
		}
	]
}
```

### Type `created_team`

Generated when creating teams.

This activity contains the following fields:
- "team_id": unique ID of the created team.
- "team_name": the name of the created team.

#### Example

```json
{
	"team_id": 123, 
	"team_name": "foo"
}
```

### Type `deleted_team`

Generated when deleting teams.

This activity contains the following fields:
- "team_id": unique ID of the deleted team.
- "team_name": the name of the deleted team.

#### Example

```json
{
	"team_id": 123, 
	"team_name": "foo"
}
```

### Type `applied_spec_team`

Generated when applying team specs.

This activity contains a field "teams" where each item contains the team details with the following fields:
- "id": Unique ID of the team.
- "name": Name of the team.

#### Example

```json
{
	"teams": [
		{
			"id": 123, 
			"name": "foo"
		}
	]
}
```

### Type `edited_agent_options`

Generated when agent options are edited (either globally or for a team).

This activity contains the following fields:
- "global": "true" if the user updated the global agent options, "false" if the agent options of a team were updated.
- "team_id": unique ID of the team for which the agent options were updated (null if global is true).
- "team_name": the name of the team for which the agent options were updated (null if global is true).

#### Example

```json
{
	"team_id": 123, 
	"team_name": "foo",
	"global": false
}
```

### Type `live_query`

Generated when running live queries.

This activity contains the following fields:
- "targets_count": Number of hosts where the live query was targeted to run.
- "query_sql": The SQL query to run on hosts.
- "query_name": Name of the query (this field is not set if this was not a saved query).

#### Example

```json
{
	"targets_count": 5000, 
	"query_sql": "SELECT * from osquery_info;",
	"query_name": "foo"
}
```

### Type `user_added_by_sso`

Generated when new users are added via SSO JIT provisioning

This activity does not contain any detail fields.



<meta name="pageOrderInSection" value="1400">