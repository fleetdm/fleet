// "Review the AI" quiz: click the line where the AI-drafted code is wrong.
// Each question embodies a Fleet convention AI commonly fumbles. Grounded in
// CLAUDE.md explicit rules and personal-memory feedback corrections.

export type ReviewAIArea = "frontend" | "backend";

export type ReviewAIQuestion = {
  id: string;
  area: ReviewAIArea;
  title: string;
  context: string;
  language: "tsx" | "ts" | "go" | "sql" | "yaml";
  code: string;
  flaggedLines: number[];
  mistake: string;
  fix: string;
  conventionLink?: { label: string; href: string };
};

export const REVIEW_AI_QUESTIONS: ReviewAIQuestion[] = [
  {
    id: "handrolled-modal",
    area: "frontend",
    title: "Delete confirmation modal",
    context:
      "AI is asked to add a confirmation modal before deleting a host. Click the line that first signals a coding convention mistake.",
    language: "tsx",
    code: `const baseClass = "delete-host-confirmation-modal";

export const DeleteHostConfirmationModal = ({ hostName, onConfirm, onCancel }: Props) => {
  return (
    <div className={\`\${baseClass}__backdrop\`}>
      <div className={\`\${baseClass}__dialog\`}>
        <h2 className={\`\${baseClass}__title\`}>Delete {hostName}?</h2>
        <p className={\`\${baseClass}__message\`}>This cannot be undone.</p>
        <div className={\`\${baseClass}__actions\`}>
          <button
            onClick={onCancel}
            className={\`\${baseClass}__cancel-button\`}
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={\`\${baseClass}__delete-button\`}
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
};`,
    flaggedLines: [5, 6, 10, 16],
    mistake:
      "Hand-rolled modal + backdrop + custom buttons when Fleet already has a Modal component and a Button component. The `__backdrop` div is the smoking gun. If you're inventing a backdrop class, you're reimplementing Modal from scratch, which means losing focus trapping, keyboard dismissal, scroll-lock, and the Fleet visual language all at once. Raw `<button>` elements with hand-rolled classNames are the same mistake one level down.",
    fix: "Use `components/Modal` and wrap the two actions in `components/Button` with `variant=\"alert\"` for Delete and `variant=\"inverse\"` for Cancel. The baseClass + `__element` pattern is right for Fleet; it's just the wrong level of the component tree.",
    conventionLink: {
      label: "frontend/components",
      href: "https://github.com/fleetdm/fleet/tree/main/frontend/components",
    },
  },
  {
    id: "free-check-inverted",
    area: "frontend",
    title: "Fleet Free upgrade banner",
    context:
      "AI is asked to show an upgrade banner to Fleet Free users on the policies page. Click the line that first signals a coding convention mistake.",
    language: "tsx",
    code: `export const PoliciesUpgradeBanner = ({ isPremiumTier }: Props) => {
  if (isPremiumTier) {
    return (
      <Banner variant="info">
        Fleet Free users can upgrade to unlock scheduled policy runs.
      </Banner>
    );
  }
  return null;
};`,
    flaggedLines: [2],
    mistake:
      "`isPremiumTier` returns true for Premium users. This banner is for Free users, so the condition should be `!isPremiumTier`. The negation is load-bearing: miss it and the banner shows to the wrong tier.",
    fix: "`if (!isPremiumTier)`. Shorthand like `isPremiumTier && …` would also hide the banner from the people it's meant for.",
    conventionLink: {
      label: "Memory: Fleet Free check",
      href: "https://github.com/fleetdm/fleet",
    },
  },
  {
    id: "missing-premium-gate",
    area: "backend",
    title: "New enterprise endpoint",
    context:
      "AI is asked to add a service method for a new premium-only feature (bulk configure VPP apps). Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (svc *Service) BulkConfigureVPPApps(ctx context.Context, teamID uint, apps []fleet.VPPAppConfig) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{ID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, app := range apps {
		if err := svc.ds.UpsertVPPAppConfig(ctx, teamID, app); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting VPP app config")
		}
	}
	return nil
}`,
    flaggedLines: [2, 7],
    mistake:
      "The first operation in the method body is an authz check. No license check before it. The subsequent write (`svc.ds.UpsertVPPAppConfig`) runs for anyone who passes authz, regardless of tier. For an enterprise feature, a license check must come BEFORE authz. Without it, Fleet Free users with team-write permission can silently use a premium feature.",
    fix: "Prepend a license guard: `if !fleet.IsPremium(ctx) { return fleet.ErrMissingLicense }` (or `NewLicenseErrorWithCause`). Enterprise service wrappers in `ee/server/service/` are the canonical place for this.",
    conventionLink: {
      label: "ee/server/service patterns",
      href: "https://github.com/fleetdm/fleet/tree/main/ee/server/service",
    },
  },
  {
    id: "upsert-self-collision",
    area: "backend",
    title: "Policy name uniqueness check",
    context:
      "AI is asked to add a uniqueness check so two policies in the same team can't share a name. The code works on insert but breaks on update. Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (ds *Datastore) savePolicy(ctx context.Context, p *fleet.Policy) error {
	var exists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &exists, ` + "`" + `
		SELECT EXISTS(
			SELECT 1 FROM policies
			WHERE team_id = ? AND name = ?
		)
	` + "`" + `, p.TeamID, p.Name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking policy name uniqueness")
	}
	if exists {
		return &fleet.ConflictError{Message: "policy name already exists for this team"}
	}
	return ds.upsertPolicy(ctx, p)
}`,
    flaggedLines: [6, 12, 13],
    mistake:
      "The uniqueness check doesn't exclude the policy being updated. On insert (new row, no id) it works. On update (existing row), the row's own `(team_id, name)` collides with itself: the EXISTS clause fires, `if exists` passes, and a false ConflictError returns. The missing `AND id <> ?` on the WHERE is the fix location; the `if exists` branch is where the false positive manifests.",
    fix: "Add `AND id <> ?` with the policy's id (or omit the check entirely for the insert vs. update path and use a unique index + ON DUPLICATE KEY handling). Check the surrounding 100 lines of other `saveX` methods. This pattern is already solved nearby.",
    conventionLink: {
      label: "CLAUDE.md: duplicate checks",
      href: "https://github.com/fleetdm/fleet/blob/main/.claude/CLAUDE.md",
    },
  },
  {
    id: "pagination-not-reset",
    area: "frontend",
    title: "Hosts page status filter",
    context:
      "AI is asked to add a status filter to the hosts page. Filtering works, but after switching from \"all\" to \"online\" while on page 4, the table shows an empty page. Click the line that first signals a coding convention mistake.",
    language: "tsx",
    code: `const HostsPage = () => {
  const [statusFilter, setStatusFilter] = useState<HostStatus>("all");
  const [pageIndex, setPageIndex] = useState(0);

  const { data: hostsData } = useQuery(
    ["hosts", statusFilter, pageIndex],
    () => hostsAPI.loadAll({ status: statusFilter, page: pageIndex }),
  );

  const handleStatusFilterChange = (next: HostStatus) => {
    setStatusFilter(next);
  };

  return (
    <>
      <Dropdown
        value={statusFilter}
        onChange={handleStatusFilterChange}
        options={STATUS_OPTIONS}
      />
      <TableContainer
        data={hostsData?.hosts ?? []}
        pageIndex={pageIndex}
        onPaginationChange={setPageIndex}
      />
    </>
  );
};`,
    flaggedLines: [11, 12],
    mistake:
      "The status-filter handler updates the filter but doesn't reset `pageIndex`. If the user is on page 4 of 10 and narrows to a smaller result set, page 4 is now out of range and the table renders empty. Any state change that narrows a result set must reset pagination to 0.",
    fix: "Call `setPageIndex(0)` alongside `setStatusFilter(next)`. Same rule for search inputs, team scoping, and any other filter that changes the row count.",
    conventionLink: {
      label: "Memory: pagination reset",
      href: "https://github.com/fleetdm/fleet/tree/main/frontend",
    },
  },
  {
    id: "mutation-no-cache-invalidation",
    area: "frontend",
    title: "Hosts page. Delete confirmation flow",
    context:
      "AI is asked to wire up delete on the hosts page: table row triggers a confirmation modal, modal confirm fires the API call. The call succeeds, the modal closes, the success flash shows, but the deleted host keeps showing in the table until a hard refresh. Click the line that first signals a coding convention mistake.",
    language: "tsx",
    code: `const HostsPage = () => {
  const [modalHostId, setModalHostId] = useState<string | null>(null);
  const { data: hostsData } = useQuery(["hosts"], () => hostsAPI.loadAll());

  const handleOpenDeleteModal = (hostId: string) => {
    setModalHostId(hostId);
  };

  const handleConfirmDelete = async () => {
    if (!modalHostId) return;
    try {
      await hostsAPI.destroy(modalHostId);
      renderFlash("success", \`Host \${modalHostId} deleted\`);
      setModalHostId(null);
    } catch (error) {
      renderFlash("error", "Could not delete host. Please try again.");
    }
  };

  return (
    <>
      <HostTable
        hosts={hostsData?.hosts ?? []}
        onOpenDeleteModal={handleOpenDeleteModal}
      />
      {modalHostId && (
        <DeleteHostConfirmationModal
          hostId={modalHostId}
          onConfirm={handleConfirmDelete}
          onCancel={() => setModalHostId(null)}
        />
      )}
    </>
  );
};`,
    flaggedLines: [11, 12, 13, 14],
    mistake:
      "The page owns the `[\"hosts\"]` query AND the destroy call AND the modal state. All three successful-write side effects fire. API destroy, success flash, modal close. But nothing invalidates the cache. The server removes the host; this component's own cached list still shows it until the query refetches on its own (or the user hard-refreshes). The responsibility isn't delegated to anyone else. It ends here.",
    fix: "Use `useMutation` with `onSuccess` → `queryClient.invalidateQueries(['hosts'])`. The async-handler + try/catch shape is the pattern that most often loses the invalidation step because there's no natural hook for \"what to do when the write succeeds.\" `useMutation`'s `onSuccess` makes the right thing the easy thing.",
    conventionLink: {
      label: "Memory: useMutation for replica lag",
      href: "https://github.com/fleetdm/fleet/tree/main/frontend",
    },
  },
  {
    id: "missing-batch-entry-point",
    area: "backend",
    title: "GitOps user spec validation",
    context:
      "AI just added email format validation to the single `InviteUser` endpoint to fix a bug where malformed emails got stored. This is the GitOps batch path that was NOT updated. Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (svc *Service) ApplyUsersSpec(ctx context.Context, spec []fleet.UserSpec) error {
	if !fleet.IsPremium(ctx) {
		return fleet.ErrMissingLicense
	}
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionWrite); err != nil {
		return err
	}
	for _, u := range spec {
		if err := svc.ds.CreateOrUpdateUserFromSpec(ctx, u.Email, u.Name, u.Role); err != nil {
			return ctxerr.Wrap(ctx, err, "creating user from gitops spec")
		}
	}
	return nil
}`,
    flaggedLines: [8, 9],
    mistake:
      "The single-user path got email validation; this batch/GitOps path didn't. The datastore call accepts whatever `u.Email` holds, so GitOps replays can still store malformed emails. Fixing the symptom in one entry point and not the other is a classic Fleet trap.",
    fix: "Add the same validation call (`fleet.ValidateEmail(u.Email)` or whatever the InviteUser path uses) immediately before the datastore call. Rule: when fixing an input-handling bug, grep for every entry point to the operation (single, batch, GitOps, fleetctl) and cover them all.",
    conventionLink: {
      label: "CLAUDE.md: cover all entry points",
      href: "https://github.com/fleetdm/fleet/blob/main/.claude/CLAUDE.md",
    },
  },
  {
    id: "fix-at-reproduction-step",
    area: "backend",
    title: "Policy iteration counter",
    context:
      "Bug report: policy failure counts in the API response are one iteration ahead of reality. AI drafts this patch to the read endpoint. Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (svc *Service) GetPolicyReport(ctx context.Context, policyID uint) (*fleet.PolicyReport, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{ID: policyID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	report, err := svc.ds.GetPolicyReport(ctx, policyID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading policy report")
	}
	// Reports lag one iteration behind the evaluation service. Subtract 1.
	report.IterationCount = report.IterationCount - 1
	return report, nil
}`,
    flaggedLines: [9, 10],
    mistake:
      "The fix is at the reproduction step (the read endpoint) instead of at the root cause (the evaluation service that increments the counter at the wrong moment). The hack line (`- 1`) is the fix location; the comment above it is a tell. When the code needs a comment explaining why it's doing something inconsistent, that's usually a signal the fix is in the wrong place. The `-1` will drift the moment anything else reads the same counter, and will be forgotten the next time someone touches this file.",
    fix: "Fix the evaluation service to increment the counter when the check actually runs, not when results come back. Rule: identify WHERE in the request lifecycle the problem manifests. Fix it there, not where it's visible.",
    conventionLink: {
      label: "CLAUDE.md: fix at the right place",
      href: "https://github.com/fleetdm/fleet/blob/main/.claude/CLAUDE.md",
    },
  },
  {
    id: "declarative-validated-against-db",
    area: "backend",
    title: "GitOps policies spec apply",
    context:
      "AI is asked to add a conflict check to the GitOps policies-apply path so two policies in the same team can't share a name. This fires false conflicts on replay. Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (svc *Service) ApplyPoliciesSpec(ctx context.Context, specs []fleet.PolicySpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.Policy{}, fleet.ActionWrite); err != nil {
		return err
	}
	for _, p := range specs {
		existing, err := svc.ds.PolicyByName(ctx, p.TeamID, p.Name)
		if err == nil && existing != nil {
			return &fleet.ConflictError{Message: fmt.Sprintf("policy %q already exists", p.Name)}
		}
		if err := svc.ds.UpsertPolicySpec(ctx, p); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting policy spec")
		}
	}
	return nil
}`,
    flaggedLines: [6, 7, 8],
    mistake:
      "GitOps is declarative: the incoming spec IS the source of truth. Consulting the DB for \"does this name already exist\" treats the DB as source-of-truth, so every replay of a spec that mentions existing policies fires a false conflict. The DB lookup is the fix location; the `if existing != nil` branch and the `ConflictError` return are where the false conflict manifests. The uniqueness constraint belongs within the incoming spec (duplicate names between rows), not against the DB.",
    fix: "Remove the DB lookup. Validate for duplicate `(TeamID, Name)` pairs WITHIN the `specs` slice before the loop. The datastore `UpsertPolicySpec` already handles the \"exists or doesn't\" question.",
    conventionLink: {
      label: "CLAUDE.md: declarative/batch endpoints",
      href: "https://github.com/fleetdm/fleet/blob/main/.claude/CLAUDE.md",
    },
  },
  {
    id: "gitops-policy-platform",
    area: "backend",
    title: "GitOps policy. FileVault check",
    context:
      "AI is asked to add a Fleet GitOps policy that fails any Mac or Linux laptop without full-disk encryption. The YAML parses locally, but `fleetctl gitops` rejects the apply. Click the line that first signals a coding convention mistake.",
    language: "yaml",
    code: `- name: Disk encryption is enabled
  query: SELECT 1 FROM disk_encryption WHERE encrypted = 1;
  description: Checks whether the system disk is encrypted.
  resolution: Enable full-disk encryption in system settings.
  platform:
    - darwin
    - linux
  critical: true
  team: Workstations`,
    flaggedLines: [5, 6, 7],
    mistake:
      "`platform:` is written as a YAML list, but Fleet's `PolicySpec.Platform` is a Go `string` (the field comment in `server/fleet/policies.go` says \"comma-separated string to indicate the target platforms\"). YAML list syntax fails to unmarshal into a string, so `fleetctl gitops` rejects the apply with a type error before any platform validation even runs. YAML conventions say \"use a list for multi-value\". Fleet doesn't.",
    fix: "`platform: darwin,linux`. Comma-separated, no spaces required. `Platform` is a `string` field on `PolicySpec` (unusual for a multi-value field. Most Fleet slice fields like `labels_include_any` really are YAML lists). The lesson: don't let YAML's \"use a list for multi-value\" instinct override the Go struct tag. Check the field type in `server/fleet/policies.go` before trusting the natural YAML form.",
    conventionLink: {
      label: "Fleet GitOps policy spec",
      href: "https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files/policies.md",
    },
  },
  {
    id: "missing-ctxerr-wrap",
    area: "backend",
    title: "Datastore read method",
    context:
      "AI is asked to add a datastore method to fetch a host by hardware UUID. The code compiles and works, but Fleet convention is being broken. Click the line that first signals a coding convention mistake.",
    language: "go",
    code: `func (ds *Datastore) HostByHardwareUUID(ctx context.Context, uuid string) (*fleet.Host, error) {
	var host fleet.Host
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host,
		"SELECT id, hostname, team_id, hardware_uuid FROM hosts WHERE hardware_uuid = ?",
		uuid,
	)
	if err != nil {
		return nil, err
	}
	return &host, nil
}`,
    flaggedLines: [8],
    mistake:
      "The error is returned bare instead of wrapped with `ctxerr.Wrap`. When this surfaces upstream, the stack trace won't name this call site, which makes production debugging a scavenger hunt. Every datastore method in Fleet wraps its errors.",
    fix: "`return nil, ctxerr.Wrap(ctx, err, \"getting host by hardware uuid\")`. The message should describe the operation, not restate the function name.",
    conventionLink: {
      label: ".claude/rules/fleet-go-backend.md",
      href: "https://github.com/fleetdm/fleet/blob/main/.claude/rules/fleet-go-backend.md",
    },
  },
];
