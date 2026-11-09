# Fleet REST API conventions

These patterns were mined from `docs/REST API/rest-api.md` on 2026-10-06 (552 documented routes). The counts show how strong each convention is. Re-mine before trusting a count for a close call, using the commands at the bottom.

**Majority** means follow it. **Split** means existing endpoints disagree: flag it for the API design DRI and don't enforce either side.

## URL

- **Majority**: every route is `/api/v1/fleet/<resource>[/...]`. The docs always show `v1`. In code, new routes use `ue.StartingAtVersion("2022-04")` (see `.claude/rules/fleet-api.md`).
- **Majority**: resources are plural snake_case nouns (`configuration_profiles`, `certificate_authorities`, `custom_host_vitals`). Exceptions are singletons (`config`, `logo`, `me`, `version`, `setup_experience`, `host_summary`).
- **Inconsistent outlier**: the only hyphenated path is `conditional-access/microsoft`, while its siblings use `conditional_access/...`. Don't copy it.
- **Exception**: SCIM paths (`/scim/Users`, `/scim/Groups`) and their camelCase JSON follow the SCIM RFC. Don't use them as a precedent.
- **Majority**: the main path parameter is `:id` (106 routes). Secondary parameters are named after their resource: `:profile_uuid`, `:software_title_id`, `:policy_id`, `:fleet_id`. UUID-keyed resources use `:<thing>_uuid`.
- **Patterns**:
  - Host-scoped resources and actions live under `/hosts/:id/...`.
  - Fleet Desktop (end-user) endpoints are `/device/:token/...` and mirror their admin counterparts.
  - GitOps/declarative endpoints are `/spec/<resource>`.
  - Bulk operations append `/batch` (`/scripts/run/batch`, `/configuration_profiles/resend/batch`).
  - Actions are `POST` to a verb sub-path (`/install`, `/uninstall`, `/resend`, `/cancel`, `/lock`, `/wipe`, `/refetch`).
  - "By name" variants exist alongside "by ID" variants (`Delete label by name`, `Delete report by name`).

## Methods and heading verbs

| Method | Heading verb | Count |
|---|---|---|
| GET | `Get` (single), `List` (collection), `Download` (file) | 75 / 39 / 4 |
| POST | `Create` / `Add` (new), or an action verb (`Run`, `Install`, `Resend`, ...) | 18 / 7 / many |
| PATCH | `Update` | 19 |
| PUT | `Update` / `Replace` (full replacement, spec, file upload) | 5 / 3 |
| DELETE | `Delete` (`Cancel` for pending work) | 29 / 2 |

- **Split**: some partial updates are `POST` ("Update ...", 9 cases). Prefer `PATCH` for a new partial update unless the neighbors on the same resource use `POST`.
- **Split**: `Create` vs `Add`. `Add` is used when attaching something to Fleet (software, certificate templates), and `Create` when Fleet makes the object.

## Parameters

Docs table: `| Name | Type | In | Description |`. `In` is one of `path`, `query`, `body`, or `form` (multipart). Required parameters start with `**Required**.`

Shared names, with their type and location (use exactly these):

| Name | Type | In | Notes |
|---|---|---|---|
| `id` | integer | path | 87 uses. A string only for non-numeric keys. |
| `fleet_id` | integer | query (GET), body (mutations) | Premium. Write "If not specified, ..." for the default (usually "Unassigned" or global). |
| `page` | integer | query | "Page number of the results to fetch." |
| `per_page` | integer | query | |
| `order_key` | string | query | List the allowed fields |
| `order_direction` | string | query | `**Requires `order_key`**.` Options `"asc"`/`"desc"`. Default `"asc"` (33/33). |
| `after` | string | query | Cursor. Needs `order_key`. |
| `query` | string | query | Free-text search ("Search query keywords. Searchable fields include ...") |
| `platform` | string | query/body | |
| `labels_include_any` / `labels_include_all` / `labels_exclude_any` | array | body/form | Label scoping on profiles, software, and so on |
| `host_ids` / `ids` | array | body | Bulk targets |
| `self_service` | boolean | body | |

- Pagination is the set `page`, `per_page`, `order_key`, `order_direction`, and `after`. A new list endpoint should support the same set as its neighbors (32–33 list endpoints have all of them).
- **Doc bug pattern**: a path parameter documented with `In` = `body` or `query` (for example, `id` on "Get certificate authority (CA)").

## Terminology: Fleets and Reports

- New parameter and field names use `fleet_id`, `fleet_name`, `fleets`, `report_id`, `reports`. The docs already mostly use these (66 `"fleet_id"` keys, and query examples use `fleet_id` 20 to 5).
- **Legacy**: `team_id` still shows up in 89 JSON examples. Don't flag existing occurrences in an unrelated change. Do flag new ones.
- In code, a renamed field keeps the old tag and adds `renameto`: `query:"team_id,optional" renameto:"fleet_id"` or `json:"team_id" renameto:"fleet_id"`. That makes both names accepted, and a request that sends both gets an `AliasConflictError` (`server/platform/endpointer/`). A brand-new field just uses the new name.
- "Query" in the product sense is now "report". `query` as a parameter means a search string or SQL text.

## Responses

- **Majority**: snake_case keys. IDs are integers. Timestamps are RFC 3339 UTC, like `"2025-11-04T00:00:00Z"` (402 of about 413).
- **List envelope**: `{"<plural resource>": [...], "meta": {"has_next_results": bool, "has_previous_results": bool}}`. Some lists also or instead return `"count"` (34 uses). Match the neighbors on whether `count` is included alongside `meta`.
- **Split**: whether a single object is returned bare (`{"id": 1, ...}`) or wrapped (`{"host": {...}}`). Match other endpoints on the **same resource**. For a new resource, flag it as a DRI call.
- **Watch**: a foreign-key ID returned as a string (`"certificate_authority_id": "1"` in "List certificate templates") is an inconsistency.
- **Errors** use the standard body `{"message": "...", "errors": [{"name": "<field or base>", "reason": "..."}], "uuid": "..."}`. Don't invent another shape.

## Status codes

| Code | When | Count |
|---|---|---|
| 200 | Default success, including most creates | 251 |
| 202 | A queued or async action on a host (install, resend, run script, refetch) | 15 |
| 204 | Success with no body | 22 |
| 201 | SCIM creates only. Fleet creates return 200. | 2 |
| 409 | Name or uniqueness conflict, or a conflicting state | 3 |
| 422 | Validation failure | 3 |
| 404 | Not found | 4 |

- **Split**: DELETE returns 200 (19, mostly older endpoints: hosts, users, labels, fleets, reports) or 204 (14, mostly newer: scripts, software, certificates, assets, CAs). Expect new endpoints to use 204, but flag it as a DRI call if the resource's siblings use 200.

## Premium, auth, and registration

- Premium-only endpoints and parameters start their description with `_Available in Fleet Premium._` (the punctuation varies: 126 have a trailing `.`, 70 and 51 use other forms. Prefer `_Available in Fleet Premium._`).
- Non-default auth (Android node key, Fleet Desktop token) gets a `#### Request headers` section showing each accepted `Authorization` form.
- A request body that can be larger than `FLEET_SERVER_DEFAULT_MAX_REQUEST_BODY_SIZE` (1 MiB) states its limit, and the code uses `ue.WithRequestBodySizeLimit`.
- Every new endpoint goes in `server/api_endpoints/api_endpoints.yml` (`method`, `path` with `:param`, and a `display_name` that matches the doc heading). This drives granular permissions for API-only users (handbook: `handbook/product-design/README.md`).

## Doc section structure

```
### <Verb> <resource>
<One-sentence description.>
`METHOD /api/v1/fleet/...`
#### Parameters
| Name | Type | In | Description |
#### Example
`METHOD /api/v1/fleet/...` (concrete IDs, e.g. `/hosts/123`)
##### Request body          (when there is one)
##### Default response
`Status: NNN`
```json ... ```
```

Use `#### Example (<variant>)` for multiple examples. Add the endpoint to the section's TOC list near the top of its `##` section.

## Re-mining

```bash
F="docs/REST API/rest-api.md"
# Methods
grep -oE '^`(GET|POST|PATCH|PUT|DELETE) /api/[^`]+`' "$F" | awk '{print $1}' | sort | uniq -c
# Parameter name/type/in
grep -E '^\| *[a-z_]+ *\| *[a-z]+' "$F" | awk -F'|' '{gsub(/ /,"",$2);gsub(/ /,"",$3);gsub(/ /,"",$4);print $2,$3,$4}' | sort | uniq -c | sort -rn | head -40
# First status code per endpoint, by method
awk '/^### /{h=$0;m=""} /^`(GET|POST|PATCH|PUT|DELETE) \/api/ && m==""{split($1,a,"`");m=a[2]} /^`Status: [0-9]+/ && m!="" && !d[h]++{match($0,/[0-9]+/);print m,substr($0,RSTART,RLENGTH)}' "$F" | sort | uniq -c
```
