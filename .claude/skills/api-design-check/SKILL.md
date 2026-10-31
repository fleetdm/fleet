---
name: api-design-check
description: Check a proposed or changed Fleet REST API endpoint against existing API conventions (URL shape, parameter names, pagination, response envelope, status codes, Fleets/Reports naming, Premium markers, api_endpoints.yml). Use when asked to "check API conventions", "review this API design", "is this endpoint consistent", or when reviewing a reference-docs PR labeled ~api-or-yaml-design.
allowed-tools: Read, Grep, Glob, Bash
effort: high
---

# Check an API design against existing Fleet conventions

Target: $ARGUMENTS

The target can be a PR number, a branch, a spec file, or a pasted endpoint. With no target, use the current branch's diff against `main`.

Fleet has no written API style guide. The rule is "be consistent with what's already there", so this check compares the proposal with **real existing endpoints**. `references/conventions.md` records the patterns that are hard to infer from a single neighbor.

## 1. Collect the proposed endpoints

- **Docs PR or branch**: diff `docs/API/rest-api.md` (and `docs/Contributing/reference/api-for-contributors.md` for internal endpoints). Each endpoint is a `### Heading`, then a `` `METHOD /api/v1/fleet/...` `` line, then `#### Parameters`, `#### Example`, and `##### Default response`.
- **Code**: diff `server/service/handler.go` and `ee/server/service/handler.go` for new `ue.GET/POST/PATCH/PUT/DELETE` registrations. Read the matching request and response structs for the parameter and response field names.
- **Spec or pasted text**: read the endpoint definitions directly.

For each endpoint, write down: the method, the path, the parameters (name, type, `in`), the response body keys, the status code, and whether it's Premium.

## 2. Find the closest existing endpoints

For each proposed endpoint, find 2–4 existing endpoints to compare it with, in `docs/API/rest-api.md`. Use them in this order:

1. The same resource with a different verb. For a new `PATCH /fleet/foo/:id`, compare `GET /fleet/foo/:id` and `POST /fleet/foo`.
2. The same kind of operation on a sibling resource. A new list endpoint is compared with other `List ...` endpoints, a host-scoped action with other `POST /hosts/:id/...` actions, and a Fleet Desktop endpoint with other `/device/:token/...` endpoints.
3. Recently added endpoints over old ones when they disagree. Check with `git log -S '<path>' --format='%h %ad' --date=short -- "docs/API/rest-api.md"`. Older endpoints are often the inconsistent ones.

Quote the neighbor you compared against in each finding. A finding with no existing endpoint as evidence is an opinion, so drop it or label it as one.

## 3. Check each dimension

Go through `references/conventions.md` section by section. At minimum, check these:

- **URL**: the resource noun and its plural form, snake_case, path parameter names, and where it sits in the hierarchy (`/hosts/:id/...`, `/device/:token/...`, `/spec/...`, `.../batch`).
- **Method and verb**: the HTTP method matches the heading verb (List/Get/Create/Add/Update/Delete), and actions are `POST` to a sub-path.
- **Parameters**: the names and types of shared parameters (`fleet_id`, `page`, `per_page`, `order_key`, `order_direction`, `after`, `query`, `platform`, `labels_include_any`, ...), and that `In` (path/query/body/form) is correct. A path parameter documented as `body` is a doc bug.
- **Response**: the list envelope (`{"<plural>": [...], "meta": {...}}` and/or `count`), the single-object shape (a bare object or wrapped in `{"<singular>": {...}}`, matching the resource's other endpoints), IDs as integers, timestamps as RFC 3339 UTC, and snake_case keys.
- **Status codes**: 200 for normal success, 202 for queued or async host actions, 204 for no body, and 201 only for SCIM. Error codes should match the semantics neighbors use (409 conflict, 422 validation, 404 not found).
- **Terminology**: new names use `fleet`/`fleets` and `report`/`reports`, never `team`/`query` in the product sense. Renamed fields need the `renameto` alias in code.
- **Premium and auth**: the `_Available in Fleet Premium._` marker on Premium-only parameters and endpoints, and a documented non-default auth (node key, device token).
- **Registration**: the new endpoint is listed in `server/api_endpoints/api_endpoints.yml` with a display name that matches the doc heading.
- **Doc structure**: the section order, a `Default response` with `Status:`, an example that uses realistic IDs, and a link from the section's table of contents.

## 4. Report

Group the findings like this:

1. **Inconsistent**: the proposal deviates from a clear majority pattern. Show the proposed value, the existing convention, and 1–2 neighbor endpoints as evidence.
2. **Split convention**: existing endpoints disagree (`conventions.md` marks these). Don't pick a winner. Name both patterns and say that it's a call for the API design DRI.
3. **Doc nits**: wrong `In` column, a missing `Status:` line, an anchor not in the TOC, and so on.

Keep it short: one line per finding plus the evidence. If a dimension is consistent, don't list it. End by saying which endpoints were compared.

Don't edit files or post on GitHub. This is a read-only check.
