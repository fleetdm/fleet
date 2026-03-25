# Use OpenAPI YAML specs for defining Fleet's web APIs

## Status

Proposed

## Date

2026-03-25

## Context

Fleet's REST API is one of the most heavily used interfaces in the product. It is consumed by the Fleet UI, `fleetctl` CLI, GitOps workflows, customer integrations, AI agents, and third-party tooling. The API is a first-class product surface, and investing in how it is specified and documented is a direct investment in Fleet's quality and user experience.

As Fleet's API surface, customer base, and ecosystem integrations grow, the current Markdown-based approach creates challenges that grow with Fleet's scale:

**For users and integrators**, the absence of a machine-readable spec means there is no trivially machine-readable way to auto-generate client SDKs, import Fleet's API into tools like [Postman](https://www.postman.com), [Insomnia](https://insomnia.rest), or [Bruno](https://www.usebruno.com), or provide AI agents and LLM-powered workflows with a reliable, structured description of the API's capabilities. Customers have already requested an OpenAPI spec ([#18744](https://github.com/fleetdm/fleet/issues/18744)), noting that a published spec would make it significantly easier to build integrations against Fleet. The absence of a spec has been a friction point for adoption, particularly for those building integrations and AI-powered workflows on top of Fleet's API.

Easily integrating automated workflows into Fleet — where machines, not humans, are the primary API consumers — creates stickier, more valuable deployments. Customers who can build reliable automation on top of Fleet are less likely to evaluate alternatives. A machine-readable spec is the foundation that makes this possible at scale.

**For contributors**, there is often confusion that leads to avoidable back-and-forth during PRs, because the current Markdown docs aren't comprehensive enough to fully specify the API contract. When implementing a new endpoint or writing client code against an existing one, contributors often need to ask questions mid-implementation or read the Go source directly. This means Fleet's API surface (which must survive backwards compatibility guarantees) doesn't get the same level of design rigor as visual wireframes, which can be tweaked without breaking compatibility.

**For product designers**, the API design review process (currently represented by the `~api-or-yaml-design` label and PRs against docs release branches) requires opening a PR to Markdown documentation rather than to a structured, machine-readable contract. This makes it harder to reason about the full shape of an API change or validate that the proposed design is internally consistent before implementation begins. Because Markdown docs aren't fully comprehensive, larger-scale API edits can result in items being missed, rendering docs silently inaccurate over time.

**For documentation maintainers**, the REST API reference docs at `fleetdm.com/docs/rest-api` are written and updated by hand. Documentation drift (where the live docs diverge from the actual implementation) is something that can happen gradually and is difficult to detect automatically. In practice, this has occasionally created issues where integrators built against documented behavior that didn't match reality.

[We anticipated the need for a more declarative format to become the source of truth](https://fleetdm.com/handbook/company/why-this-way#why-not-continuously-generate-rest-api-reference-docs-from-javadoc-style-code-comments). Importantly, this proposal is not about generating docs from code comments (the approach the handbook cautions against), but about a standalone, human-authored YAML spec reviewed independently of the implementation, just like Fleet's [wireframe-first approach](https://fleetdm.com/handbook/company/why-this-way#why-do-we-use-a-wireframe-first-approach) to product design.

OpenAPI is the widely adopted industry standard for describing HTTP APIs in a structured, machine-readable YAML or JSON format. It is supported by a broad ecosystem of tooling for documentation generation, client SDK generation, linting, mocking, and contract testing. Other companies with comparable open-source API-first products, including GitLab, Stripe, and Fastly, use OpenAPI as the authoritative source of truth for their API contracts. Fleet already uses YAML extensively as a first-class interface for GitOps configuration, making OpenAPI YAML a natural fit.

## Decision

Fleet will adopt OpenAPI YAML specs as the authoritative definition for its web APIs.

Specs will be stored in the `fleetdm/fleet` repository under `docs/api/` and managed under version control like any other source artifact.

The OpenAPI spec will serve as:

1. **The source of truth for API design review.** Product designers and contributors will propose API changes by opening pull requests to the OpenAPI spec first, before implementation begins. The `~api-or-yaml-design` review step will operate against the spec rather than against hand-written Markdown docs.

2. **The input for generated reference documentation.** The hand-maintained REST API reference on `fleetdm.com/docs/rest-api` will be replaced with docs generated from the spec, guarding against documentation drift. The spec itself will also be publicly linked from the docs so that integrators can point their tooling directly at it.

3. **A published artifact for integrators and AI agents.** The spec will be made publicly available so that Fleet users, third-party tool authors, and AI agents can understand Fleet's API surface, generate client code, and build integrations reliably. LLM-based tooling can consume the spec directly to call the API correctly and efficiently — reducing the risk of unpredictable behavior from clients built against incomplete or inconsistent documentation.

Additionally, the spec opens the door to [Arazzo](https://spec.openapis.org/arazzo/latest.html), a companion standard that builds on OpenAPI to describe multi-step API workflows. This would allow Fleet to formally specify common workflows that require multiple API calls — in a way that both humans and AI agents can reliably follow.

All existing endpoints will be migrated to the spec in a single dedicated sprint, prioritizing the complete REST API surface over incremental coverage. Partial spec coverage is worse than no spec, as it creates ambiguity about which endpoints are authoritative and which are not. Once the initial migration is complete, product design is responsible for keeping the spec current for all new endpoints.

## Consequences

### Positive

**For users and integrators:**
- External developers building on the Fleet API gain a machine-readable contract they can use to generate typed client libraries, validate responses, and stay ahead of breaking changes.
- Tools like Postman, Insomnia, Bruno, Redoc, Swagger UI, and VS Code extensions can consume the spec directly, reducing the barrier to exploring and integrating with Fleet.

**For AI agents and LLM-powered tooling:**
- AI agents and LLM-based workflows that need to interact with Fleet's API can consume an OpenAPI spec directly to understand available endpoints, required parameters, authentication, and response shapes, without having to infer behavior from Markdown documentation or reverse-engineer the implementation.
- As AI-powered IT and security tooling grows, Fleet's API will increasingly be called by agents acting on behalf of users. A well-structured, machine-readable spec is the most reliable way to ensure those agents call the API correctly and efficiently — avoiding the performance and correctness issues that arise when clients are built against incomplete documentation.
- The spec can be fed directly into LLMs to accelerate contributor onboarding and development. For example, a contributor can bootstrap familiarity with Fleet's API surface by asking an LLM to explain or work with the spec, reducing the learning curve without requiring deep upfront reading.
- Looking ahead, [Arazzo](https://spec.openapis.org/arazzo/latest.html) allows Fleet to describe multi-step API workflows on top of the OpenAPI spec, giving AI agents a structured way to understand and execute processes that span multiple API calls.

**For product and engineering velocity:**
- API design reviews become faster and more focused. Reviewers evaluate a structured contract rather than Markdown, making it easier to spot inconsistencies (e.g., parameter naming divergence, missing error codes, inconsistent pagination shapes) before implementation.
- Contributors implementing a new endpoint have a comprehensive, testable contract to build against, reducing ambiguity and the number of questions that arise mid-implementation.
- The spec can be imported into Postman, Insomnia, Bruno, or similar tools by any contributor or QA, enabling fast manual testing without hand-crafting requests.
- Mock servers can be generated directly from the spec, enabling frontend development to proceed in parallel with backend implementation rather than blocking on it. This eliminates the time contributors currently spend hand-rolling mock responses or waiting for a real endpoint to be available, and significantly reduces integration friction when the actual backend implementation lands, because the frontend was already built against an accurate contract rather than an approximation.

**For documentation quality:**
- The reference documentation is generated from the spec, guarding against the documentation drift that can occur with hand-maintained docs.
- The spec enforces a consistent structure for all endpoints: descriptions, parameter types, required vs. optional fields, and example responses are defined in a standard schema, not left to the discretion of individual contributors.
- Changelogs and deprecation notices can be tracked in the spec itself, giving integrators a single place to understand what changed between Fleet versions.

### Negative

- **Migration cost.** Migrating all existing endpoints to the spec in a single sprint is a significant investment of contributor time. This should be planned and resourced accordingly.
- **Maintenance discipline required.** The spec is only as good as the process that keeps it updated. If contributors are not diligent about updating the spec alongside implementation changes, drift can re-emerge. This requires clear ownership (a DRI for the spec) and CI enforcement, which will be introduced in a subsequent iteration after the initial migration is complete.
- **Learning curve.** Contributors unfamiliar with OpenAPI will need to ramp up on spec syntax and conventions. Initial PRs may require more review cycles until team-wide fluency is established. Note that LLMs can help bootstrap this learning curve by generating an initial draft spec or explaining conventions on demand.
- **Risk of a half-finished migration.** Running a parallel implementation (e.g. generated types alongside the existing `endpointer` pattern) risks creating a permanent dual-stack that never fully resolves. The migration plan below is specifically designed to avoid this by committing to a complete migration in a single sprint rather than an open-ended incremental approach.

## Alternatives considered

### 1. Continue with hand-written Markdown documentation

The current approach: contributors write and maintain the REST API reference in `rest-api.md` by hand, with API design reviewed via PRs to that document.

**Pros:** No new tooling or process investment required. Contributors already know the format. No migration cost. This approach has served Fleet well through significant growth and is unlikely to cause acute failures in the near term.

**Cons:** There is no trivially machine-readable contract, so integrators, AI agents, and tooling cannot reliably consume the API programmatically. Inconsistencies across endpoints are hard to catch systematically. This approach becomes harder to sustain as the API surface grows and customer integration expectations increase.

**Why not selected:** Fleet's growth in API surface, customer base, and ecosystem integrations has reached a point where a machine-readable spec delivers compounding returns that hand-written docs cannot. A spec-first approach better positions Fleet to meet the expectations of an expanding customer base and a growing ecosystem of tools and AI agents, while reducing the maintenance burden on contributors as the API continues to grow.

---

### 2. Generate docs from Javadoc-style code comments

Annotate Go handler code with structured comments and use a tool to extract an OpenAPI spec or documentation directly from the source.

**Pros:** The spec stays colocated with the implementation, reducing the risk of them diverging. No separate spec file to maintain.

**Cons:** Fleet's own handbook explicitly cautions against this approach, noting that autogenerating docs from code comments is not always the best way to keep reference docs accurate. Documentation becomes siloed in the codebase, raising the barrier to contribution for non-contributors. In practice, tends to produce documentation that reflects implementation details rather than a thoughtfully designed user-facing contract. API design review happens after code is written, not before.

**Why not selected:** This approach inverts the design-first intent of this ADR. The goal is to make the spec the input to implementation, not an output from it. It also conflicts with Fleet's existing handbook guidance and Fleet's wireframe-first approach to product design.

---

### 3. Adopt a fully generated server implementation (e.g. ogen) immediately

Replace Fleet's existing `endpointer`-based handler pattern with a fully generated server interface derived from the OpenAPI spec, in one step.

**Pros:** Maximum consistency between spec and implementation. Eliminates an entire category of drift. Strong type safety enforced by the generator.

**Cons:** Fleet's `endpointer` abstraction and `fleet.Service` interface do not map cleanly to the server interfaces generated by tools like ogen. A full cutover would require a significant rewrite of the handler layer, middleware, and auth flow simultaneously. This is out of scope for this iteration, which treats the spec purely as a documentation and design artifact, independent of the Go implementation.

**Why not selected:** Any changes to the Go codebase are out of scope for this iteration. The value of a machine-readable spec is largely independent of whether the server is generated from it. Code generation can be revisited as a future iteration once the spec is stable and well-established.

## Migration plan

Migration is intended to be comprehensive rather than incremental. Partial spec coverage creates more problems than it solves — ambiguity about which endpoints are authoritative, arguments about prioritization, and a prolonged period where the spec cannot be trusted as a complete reference.

**Phase 1 — Evaluate and select a Markdown generation tool.** Before any spec work begins, identify and validate a tool for generating Markdown from the spec that matches Fleet's doc style and integrates cleanly with fleetdm.com. This should be timeboxed — the goal is a working pipeline, not a perfect one. Until this is validated, the spec cannot replace the hand-written docs.

**Phase 2 — Migrate all existing endpoints in a single sprint.** Once the generation pipeline is proven, allocate part of the same sprint for all teams to write spec entries for the entire REST API surface. Ownership follows the product group structure — each group owns the spec entries for their endpoints, reviewed by the API design DRI. At the end of this sprint, the hand-written `rest-api.md` is retired and the generated Markdown becomes the live docs. Any PRs not yet merged at that point are updated to include spec entries before merging.

**Phase 3 — Spec-first for new endpoints.** With the migration complete and any process issues addressed, any new endpoint requires an OpenAPI spec entry as part of the `~api-or-yaml-design` PR.

**Phase 4 — CI enforcement in a subsequent iteration.** Once the initial migration is complete, CI will be configured to fail if a new handler is added without a corresponding spec entry. This is explicitly deferred from the initial migration to avoid blocking work during the transition.

## References

- [Fleet handbook: Why not continuously generate REST API reference docs from Javadoc-style code comments?](https://fleetdm.com/handbook/company/why-this-way#why-not-continuously-generate-rest-api-reference-docs-from-javadoc-style-code-comments)
- [Fleet GitHub issue: Create OpenAPI spec for Fleet API (#18744)](https://github.com/fleetdm/fleet/issues/18744)
- [OpenAPI 3.2 specification](https://spec.openapis.org/oas/latest.html)
- [Arazzo specification](https://spec.openapis.org/arazzo/latest.html)
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
- [ogen](https://github.com/ogen-go/ogen)
- [Fleet REST API reference](https://fleetdm.com/docs/rest-api/rest-api)

---

## Example OpenAPI spec entry

The following is an illustrative OpenAPI spec entry for the existing `GET /api/v1/fleet/hosts` (List hosts) endpoint. This is what a spec-first design artifact would look like for a Fleet API endpoint — human-authored, reviewed in a PR, and used to generate both reference documentation and request/response struct boilerplate.

A few things worth noting. The `Host` schema and shared error responses are defined once under `components` and referenced with `$ref` throughout — this is how OpenAPI eliminates the duplication that currently exists across Fleet's hand-written docs. The `populate_software` parameter illustrates how a non-standard parameter shape (boolean or a specific string value) can be expressed precisely, making the contract unambiguous for both implementers and integrators. The security scheme is declared once globally, not repeated per-endpoint.

```yaml
openapi: "3.1.0"
info:
  title: "Fleet API"
  version: "1.0"

paths:
  /api/v1/fleet/hosts:
    get:
      operationId: listHosts
      summary: "List hosts"
      description: "Returns a list of hosts enrolled in Fleet, with optional filtering, sorting, and pagination. Some filters are only available in Fleet Premium."
      tags:
        - Hosts
      security:
        - bearerAuth: []
      parameters:
        - name: page
          in: query
          description: "Page number of the results to return, starting from 0."
          schema:
            type: integer
            default: 0
        - name: per_page
          in: query
          description: "Number of results to return per page. Default is 20, maximum is 500."
          schema:
            type: integer
            default: 20
            maximum: 500
        - name: order_key
          in: query
          description: "Field to sort results by. Sortable fields include `hostname`, `status`, `os_version`, `memory`, and `team_name`."
          schema:
            type: string
        - name: order_direction
          in: query
          description: "Sort direction. Either `asc` (default) or `desc`."
          schema:
            type: string
            enum: [asc, desc]
            default: asc
        - name: query
          in: query
          description: "Filter hosts by hostname, UUID, hardware serial, or primary IP address."
          schema:
            type: string
        - name: status
          in: query
          description: "Filter hosts by status. `online` hosts have checked in recently; `offline` hosts have not; `new` hosts enrolled in the last 24 hours; `missing` hosts have not been seen in 30 or more days."
          schema:
            type: string
            enum: [online, offline, new, missing]
        - name: team_id
          in: query
          description: "Filter hosts by team ID. Only available in Fleet Premium."
          schema:
            type: integer
        - name: label_id
          in: query
          description: "Filter hosts by label ID."
          schema:
            type: integer
        - name: policy_id
          in: query
          description: "Filter hosts by policy ID. Must be used with `policy_response`."
          schema:
            type: integer
        - name: policy_response
          in: query
          description: "Filter hosts by policy response. `passing` returns hosts where the policy passes; `failing` returns hosts where it fails. Requires `policy_id` to be set."
          schema:
            type: string
            enum: [passing, failing]
        - name: populate_software
          in: query
          description: "Include software inventory for each host in the response. Setting this to `true` returns significantly more data. For large fleets, consider using `without_vulnerability_details` to reduce payload size."
          schema:
            oneOf:
              - type: boolean
              - type: string
                enum: [without_vulnerability_details]
        - name: device_mapping
          in: query
          description: "Include device mapping (e.g. email addresses) for each host."
          schema:
            type: boolean
      responses:
        "200":
          description: "OK"
          content:
            application/json:
              schema:
                type: object
                properties:
                  hosts:
                    type: array
                    items:
                      $ref: "#/components/schemas/Host"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer

  responses:
    BadRequest:
      description: "Bad request: invalid or missing parameters."
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    Unauthorized:
      description: "Unauthorized: missing or invalid API token."
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
    Forbidden:
      description: "Forbidden: the authenticated user lacks the required permissions."
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

  schemas:
    Error:
      type: object
      properties:
        message:
          type: string
        errors:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
              reason:
                type: string

    Host:
      type: object
      properties:
        id:
          type: integer
        hostname:
          type: string
        uuid:
          type: string
          format: uuid
        platform:
          type: string
          example: "darwin"
        os_version:
          type: string
          example: "macOS 15.2"
        osquery_version:
          type: string
        status:
          type: string
          enum: [online, offline, new, missing]
        team_id:
          type: integer
          nullable: true
        team_name:
          type: string
          nullable: true
        memory:
          type: integer
          description: "Total memory in bytes."
        cpu_brand:
          type: string
        hardware_serial:
          type: string
        seen_time:
          type: string
          format: date-time
          description: "The last time the host contacted the Fleet server."
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
```
