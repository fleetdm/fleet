#!/usr/bin/env bash
#
# Detects frontend read-after-write anti-patterns.
#
# The pattern: after a write (POST/PATCH/DELETE), the frontend re-reads the
# same resource — via `refetchXxx()`, `queryClient.invalidateQueries(<same
# key>)`, or a follow-up GET. In production with slow read-replica lag, the
# re-read returns stale data and the user sees their save "disappear".
#
# Fix: use the write response directly. For app config, use the
# `useUpdateAppConfig` hook. For other resources, use
# `queryClient.setQueryData(key, response)` with the mutation payload.
#
# This script flags the patterns most reliably indicative of the bug. It is
# a heuristic, not a proof — cross-resource invalidations (e.g. label edit
# invalidating the hosts list) are legitimate, and sites blocked on a
# backend change are tracked in the ALLOWLIST below.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "$ROOT"

# Files intentionally exempt. Each entry needs a one-line reason.
# Prefer removing entries as the backend catches up; do not add without a
# reason a reviewer would accept.
ALLOWLIST=(
  # Backend response-shape follow-up: these writes don't return the updated
  # config today. Tracked separately for a PR that changes the endpoints.
  "frontend/pages/admin/OrgSettingsPage/cards/Info/Info.tsx"
  "frontend/pages/admin/IntegrationsPage/cards/MdmSettings/AppleMdmPage/AppleMdmPage.tsx"
  "frontend/pages/admin/IntegrationsPage/cards/ConditionalAccess/ConditionalAccess.tsx"
  "frontend/pages/ManageControlsPage/SetupExperience/cards/BootstrapPackage/BootstrapPackage.tsx"
  "frontend/pages/ManageControlsPage/SetupExperience/cards/Users/components/UsersForm/UsersForm.tsx"
  "frontend/pages/ManageControlsPage/OSSettings/cards/HostNameTemplate/HostNameTemplate.tsx"
  # Landed on main after this branch was cut. The multi-platform save
  # writes to /mdm/disk_encryption which doesn't return the full config,
  # so priming AppContext still requires a follow-up read today.
  "frontend/pages/ManageControlsPage/OSSettings/cards/DiskEncryption/DiskEncryption.tsx"

  # Design follow-up: the refetch here is load-bearing UX (form reset via
  # remount), not just cache sync. Untangle before flipping the pattern.
  "frontend/pages/ManageControlsPage/OSUpdates/components/AppleOSTargetForm/AppleOSTargetForm.tsx"
  "frontend/pages/ManageControlsPage/OSUpdates/components/WindowsTargetForm/WindowsTargetForm.tsx"
  "frontend/pages/ManageControlsPage/SetupExperience/cards/InstallSoftware/components/InstallSoftwareForm/InstallSoftwareForm.tsx"

  # List refetches over search-filtered results — splicing into the cache
  # correctly requires knowing whether the new/updated row matches the
  # active filter. Left as-is until we replace with a mutation hook.
  "frontend/pages/admin/ManageFleetsPage/TeamDetailsWrapper/UsersPage/UsersPage.tsx"

  # Legitimate cross-resource invalidations after a write. The write's OWN
  # key is primed directly; the extra invalidateQueries here targets a
  # DIFFERENT resource (hosts membership, scim_details, teams list…) whose
  # state may have changed as a side effect.
  "frontend/pages/labels/EditLabelPage/EditLabelPage.tsx"
  "frontend/pages/admin/IntegrationsPage/cards/IdentityProviders/components/GoogleWorkspaceSection/GoogleWorkspaceSection.tsx"
  "frontend/pages/admin/ManageFleetsPage/TeamDetailsWrapper/TeamDetailsWrapper.tsx"

  # Auth flows load /config directly after the auth op (signin, MFA, reset)
  # rather than through a mounted useQuery — AppContext isn't wired yet.
  # The preceding write is the auth call, not a config write, so this
  # isn't a read-after-write on the config resource.
  "frontend/components/App/App.tsx"
  "frontend/pages/MfaPage/MfaPage.tsx"
  "frontend/pages/LoginPage/LoginPage.tsx"
  "frontend/pages/ResetPasswordPage/ResetPasswordPage.tsx"

  # Mutation error handler: the request failed, so there is no fresh
  # response to prime the cache with. Invalidating is the least-bad
  # fallback until the mutation can pass through a proper error payload.
  "frontend/pages/policies/hooks/useUpdatePolicyAutomations.ts"
)

# grep -v pattern joining every allowlist entry.
BUILD_EXCLUDE() {
  local pattern=""
  for path in "${ALLOWLIST[@]}"; do
    if [[ -z "$pattern" ]]; then
      pattern="$path"
    else
      pattern="$pattern|$path"
    fi
  done
  printf "%s" "$pattern"
}
EXCLUDE_RE="$(BUILD_EXCLUDE)"

# Filters flagged lines against ALLOWLIST and test/story files.
filter() {
  grep -Ev "^($EXCLUDE_RE):" \
    | grep -Ev '\.tests?\.tsx?:' \
    | grep -Ev '\.stories\.tsx?:' \
    || true
}

echo "Scanning frontend for read-after-write patterns…"
echo

VIOLATIONS=0
REPORT=""

flag() {
  local label="$1"
  local pattern="$2"
  local hits
  hits=$(grep -rn -E "$pattern" frontend \
    --include='*.tsx' --include='*.ts' 2>/dev/null | filter || true)
  if [[ -n "$hits" ]]; then
    local count
    count=$(printf "%s\n" "$hits" | wc -l | tr -d ' ')
    VIOLATIONS=$((VIOLATIONS + count))
    REPORT+=$'\n'"[$label] $count hit(s):"$'\n'"$hits"$'\n'
  fi
}

# 1. Bare `refetchConfig()` — nearly always the app-config self-refetch.
flag "refetch-config" 'refetchConfig\s*\('

# 2. Any refetch alias that unambiguously targets an app/team config query.
flag "refetch-appconfig-alias" 'refetch(App|Team|Software|Global)Config\s*\('

# 3. Self-key invalidation of the app config after a write. This IS the
#    canonical shape of the bug; cross-resource invalidations of other keys
#    (["hosts"], ["scim_details"]) are not caught here on purpose.
flag "invalidate-config-key" 'invalidateQueries\(\s*\[\s*"config"\s*\]'

# 4. A GET /config after any prior line in the same handler — hard to
#    detect precisely without an AST, but the naked `configAPI.loadAll()`
#    call is only ever legitimate in the initial-load useQuery. Any other
#    call site is doing a manual re-read.
flag "config-load-outside-usequery" \
  'await\s+configAPI\.loadAll\(|configAPI\.loadAll\(\)\.then'

if [[ "$VIOLATIONS" -gt 0 ]]; then
  echo "$REPORT"
  echo
  echo "Detected $VIOLATIONS read-after-write pattern(s)."
  echo
  echo "After a write, don't refetch the resource you just modified — under"
  echo "read-replica lag the read returns stale data and the save appears to"
  echo "revert. Consume the write response directly:"
  echo
  echo "  For app config:"
  echo "    const updated = await configAPI.update(diff);"
  echo "    updateAppConfig(updated); // from hooks/useUpdateAppConfig"
  echo
  echo "  For any other resource:"
  echo "    const updated = await api.update(id, diff);"
  echo "    queryClient.setQueryData([key], updated);"
  echo
  echo "If the flagged site is genuinely intentional (cross-resource"
  echo "invalidation, backend returns empty, design constraint), add the"
  echo "path to ALLOWLIST in .github/scripts/detect-read-after-write.sh"
  echo "with a one-line reason."
  exit 1
fi

echo "OK: no read-after-write patterns detected."
