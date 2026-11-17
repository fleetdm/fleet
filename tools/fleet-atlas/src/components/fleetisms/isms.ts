export type FleetIsm = {
  slug: string;
  title: string;
  category:
    | "Terminology"
    | "Error handling"
    | "Authorization"
    | "Datastore"
    | "Service layer"
    | "Testing"
    | "Migrations"
    | "Review hygiene";
  rule: string; // the rule itself, 1-2 sentences
  why: string; // why this exists
  example?: string; // code sketch
  gotchas?: string[]; // common failure modes
  links?: { label: string; href: string }[];
};

export const ISMS: FleetIsm[] = [
  // ─── Terminology ───
  {
    slug: "fleets-not-teams",
    title: "Say 'fleets' and 'reports' to users; the code still says 'teams' and 'queries'",
    category: "Terminology",
    rule:
      "In new user-facing text (UI copy, docs, PRs that touch strings), use 'fleet' and 'report'. In code (variable names, table names, function names, API parameters), keep the legacy 'team' and 'query' until a coordinated rename happens.",
    why:
      "The product renamed 'teams → fleets' and 'queries → reports' but the database and APIs haven't followed yet. Renaming piecemeal breaks GitOps YAML, SDKs, and search in the codebase.",
    example:
      "// code\nvar teamID uint\n// UI copy\n'Add host to a fleet' ✓\n'Add host to a team' ✗\n// column reference\n'team_id' ✓ (still, in SQL)",
    gotchas: [
      "Writing new SQL columns called `fleet_id` to match new copy. Don't; match existing columns.",
      "Translating 'team' in a code comment to 'fleet' and leaving the variable name mismatched.",
    ],
  },

  // ─── Error handling ───
  {
    slug: "ctxerr",
    title: "Wrap errors with ctxerr before returning them",
    category: "Error handling",
    rule:
      "At every datastore and service boundary, wrap the lower-layer error: `return ctxerr.Wrap(ctx, err, \"short tag\")`. Don't return raw errors from internal functions.",
    why:
      "ctxerr attaches a stack + request context. Debug logs show the full chain of what failed and where. Raw errors give you one line and no trace.",
    example:
      'if err := ds.someOperation(ctx); err != nil {\n    return ctxerr.Wrap(ctx, err, "some operation description")\n}',
    gotchas: [
      "Logging and then returning. Pick one.",
      "Using `fmt.Errorf` instead of `ctxerr.Wrap`; loses the context.",
    ],
  },

  // ─── Authorization ───
  {
    slug: "authz-first",
    title: "First line of every protected service method: authz.Authorize",
    category: "Authorization",
    rule:
      "Open each service method with `if err := svc.authz.Authorize(ctx, target, action); err != nil { return err }` before any other work.",
    why:
      "Authz lives in Rego. If the service method starts doing lookups before authz, bugs can leak existence of entities to callers who shouldn't see them.",
    example:
      "func (svc *Service) GetReport(ctx context.Context, id uint) (*fleet.Report, error) {\n    if err := svc.authz.Authorize(ctx, &fleet.Report{ID: id}, fleet.ActionRead); err != nil {\n        return nil, err\n    }\n    // ... now do the work\n}",
    gotchas: [
      "Loading the entity before authz to populate the target struct; use IDs or the empty struct.",
      "Adding a new service method and forgetting authz entirely. There's an architecture test that scans for this.",
    ],
  },

  // ─── Datastore ───
  {
    slug: "no-sql-in-service",
    title: "Never write SQL above the datastore line",
    category: "Datastore",
    rule:
      "If the service layer needs data, add a method to the `fleet.Datastore` interface and implement it in `server/datastore/mysql/`. Services call datastore methods, not SQL.",
    why:
      "Tests swap the datastore for a mock. SQL in service code can't be mocked and can't be reused. Keeps business logic storage-agnostic.",
    gotchas: [
      "'Just this one query' is how coupling starts. Add the method.",
      "After adding a new interface method, remember to re-run `go test ./server/service/` so mocks get regenerated.",
    ],
  },
  {
    slug: "dbreader-dbwriter",
    title: "DBReader for reads, DBWriter for writes; mind replica lag",
    category: "Datastore",
    rule:
      "Reads use `ds.reader(ctx)` (replica OK). Writes use `ds.writer(ctx)` (primary). After a write, don't immediately read and expect to see it on the replica.",
    why:
      "Replica lag can be hundreds of milliseconds. 'I just saved and now I can't read it' is almost always this.",
    gotchas: [
      "Read-after-write that assumes consistency. Use the writer connection for that read, or wait, or restructure the flow.",
    ],
  },

  // ─── Service layer ───
  {
    slug: "activity-log",
    title: "Mutating service methods must call svc.NewActivity",
    category: "Service layer",
    rule:
      "Any method that mutates visible state should write an activity row via `svc.NewActivity(ctx, user, activity)` describing what happened and who did it.",
    why:
      "The activity feed is the audit log. Missing calls = holes in the trail. SOC2 cares; customers care; QA cares.",
    gotchas: [
      "Forgetting the call on new endpoints. Grep for similar activity types in the diff.",
      "Writing it before the mutation and returning an error afterward (false activity). Write after success.",
    ],
  },
  {
    slug: "ee-wrapper",
    title: "License-tier features go in ee/, not in build tags",
    category: "Service layer",
    rule:
      "Enterprise behavior lives in a struct in `ee/server/service/` that embeds the core service and overrides specific methods. No build tags; license checks are at runtime.",
    why:
      "A build-tagged binary is a different artifact. Fleet ships one binary and gates features on license at runtime.",
    example:
      "type Service struct {\n    *service.Service // embed core\n}\n\nfunc (svc *Service) FancyFeature(ctx context.Context) error {\n    if err := svc.checkLicense(ctx); err != nil { return err }\n    return svc.Service.FancyFeature(ctx)\n}",
    gotchas: [
      "Marking an ee method with `//go:build !premium`. Don't.",
    ],
  },
  {
    slug: "iterator-pattern",
    title: "Stream large reads with iterators (iter.Seq2)",
    category: "Service layer",
    rule:
      "When a method may return many rows, prefer the StreamXxx iterator variant. Range over `iter.Seq2[*fleet.T, error]`.",
    why:
      "Keeps memory bounded and lets callers cancel mid-stream via context. Go 1.23 made this ergonomic.",
    example:
      "iter, err := svc.StreamHosts(ctx, opt)\nif err != nil { return err }\nfor host, err := range iter {\n    if err != nil { return err }\n    // use host\n}",
  },

  // ─── Testing ───
  {
    slug: "cover-entry-points",
    title: "Fix bugs at every entry point: single, batch, GitOps",
    category: "Testing",
    rule:
      "If a bug exists in the single-item endpoint, check the batch endpoint and the GitOps path too. They often have parallel code.",
    why:
      "Fleet exposes most operations three ways. Fixing one leaves the other two broken and lets regressions sneak back in.",
    gotchas: [
      "Shipping a fix with a test that only covers the single-item path.",
      "Not grepping for sibling endpoints (`^.*Endpoint\\(` and `BatchX`).",
    ],
  },
  {
    slug: "validate-in-payload",
    title: "Batch endpoints validate within the incoming payload, not against the DB",
    category: "Testing",
    rule:
      "When a batch endpoint checks for duplicates or ordering constraints, verify them inside the submitted payload. Don't rely on the DB state as the only source of truth for 'this payload is consistent.'",
    why:
      "Declarative/GitOps endpoints replace state. If you only check the DB, GitOps can submit a self-inconsistent payload and succeed.",
    gotchas: [
      "Checking for duplicate names only via a `SELECT`; the duplicates were in the payload itself.",
    ],
  },

  // ─── Migrations ───
  {
    slug: "migration-ordering",
    title: "If your migration's timestamp is older than main's, bump it",
    category: "Migrations",
    rule:
      "Before merging, confirm your migration file's timestamp prefix is newer than the latest migration on main. If not, rename with the current timestamp.",
    why:
      "Goose runs migrations in filename order. An older timestamp merged after a newer one will be skipped on fresh DBs and re-run on existing ones.",
    links: [{ label: "/bump-migration skill", href: "#" }],
  },

  // ─── Review hygiene ───
  {
    slug: "no-issue-refs",
    title: "No GitHub issue refs in code or rules",
    category: "Review hygiene",
    rule:
      "Don't put `(#NNNNN)` in code, test descriptions, comments, rules, or doc one-liners. Issues drift; the ref rots.",
    why:
      "Code outlives the ticket. Context belongs in the PR description and the commit message, not the source file.",
  },
  {
    slug: "no-redundant-comments",
    title: "Comments explain WHY, not WHAT",
    category: "Review hygiene",
    rule:
      "Default to no comment. Only write one when the WHY is non-obvious: a hidden constraint, a subtle invariant, a workaround for a specific bug, behavior that would surprise a reader.",
    why:
      "Well-named identifiers already describe what the code does. Comments that restate behavior drift against the code and lie faster than code does.",
    gotchas: [
      "'Loop through all hosts' above a `for host := range hosts {`. Delete it.",
      "'Added for ticket X.' Delete it; it belongs in the PR description.",
    ],
  },
];
