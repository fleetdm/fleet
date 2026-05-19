#!/usr/bin/env python3
"""
CIS Benchmark Test Runner

Automates CIS benchmark policy testing against macOS VMs using tart,
fleetctl, and Fleet's team/MDM infrastructure.

Usage:
    python3 cis-test-runner.py --macos-version 14 --all \
        --fleet-url https://fleet.example.com --fleet-token TOKEN

Dependencies: pyyaml (pip3 install pyyaml)
"""

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
import time
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

import yaml

# Absolute temp directory, resolved relative to this file so the tool
# works from any CWD (the --cis-dir flag explicitly supports running
# outside the repo root).
TMP_DIR = Path(__file__).resolve().parent.parent.parent / "tmp"

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

VERSION_MAP = {
    "13": {
        "image": "ghcr.io/cirruslabs/macos-ventura-base:latest",
        "dir": "macos-13",
    },
    "14": {
        "image": "ghcr.io/cirruslabs/macos-sonoma-base:latest",
        "dir": "macos-14",
    },
    "15": {
        "image": "ghcr.io/cirruslabs/macos-sequoia-base:latest",
        "dir": "macos-15",
    },
    "26": {
        "image": "ghcr.io/cirruslabs/macos-tahoe-base:latest",
        "dir": "macos-26",
    },
}

SSH_OPTS = "-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR"
VM_USER = "admin"
VM_PASS = "admin"

# CIS IDs whose test scripts disable SSH, breaking our connection to
# the VM. These are tested as MANUAL (prompting the user) rather than
# running the scripts automatically.
#
# IMPORTANT: CIS section numbers are NOT stable across benchmark
# versions or between OS versions. A recommendation that is 2.3.3.4
# in macOS 14 v3.0.0 may be renumbered in the next release. Keyed by
# (os_version, cis_id).
#
# When adding a new entry, verify the mapping against the benchmark
# document for that specific OS version.
SSH_BREAKING_CIS_IDS: dict[str, set[str]] = {
    # macOS 14 Sonoma v3.0.0:
    #   2.3.3.4 - Ensure Remote Login Is Disabled (disables sshd)
    #   2.3.3.5 - Ensure Remote Management Is Disabled (also disables sshd)
    "13": set(),
    "14": {"2.3.3.4", "2.3.3.5"},
    # macOS 15 Sequoia v2.0.0:
    #   2.3.3.4 - Ensure Remote Login Is Disabled (disables sshd)
    "15": {"2.3.3.4"},
    # macOS 26 Tahoe v1.0.0:
    #   2.3.3.4 - Ensure Remote Login Is Disabled (disables sshd)
    "26": {"2.3.3.4"},
}

# CIS IDs whose MDM profiles break VM password authentication (the
# 'admin' 5-char password no longer satisfies the enforced policy, so
# SSH logins are rejected). These profiles are excluded from the bulk
# pre-test push and tested individually with a restore step.
# Keyed by OS version (CIS section numbers aren't stable across releases).
PASSWORD_POLICY_CIS_IDS: dict[str, set[str]] = {
    # macOS 14 Sonoma v3.0.0:
    #   5.2.1 - Password Account Lockout Threshold
    #   5.2.2 - Password Minimum Length (requires 15+ chars)
    #   5.2.3, 5.2.4 - Password must contain alphabetic + numeric
    #   5.2.5 - Password must contain special character
    #   5.2.6 - Password must contain uppercase+lowercase
    #   5.2.7 - Password Age
    #   5.2.8 - Password History
    "13": set(),
    "14": {"5.2.1", "5.2.2", "5.2.3", "5.2.4", "5.2.5", "5.2.6", "5.2.7", "5.2.8"},
    "15": set(),
    "26": set(),
}

# CIS IDs that cannot be reliably tested automatically due to one of:
#   - VM-specific limitations (Location Services, Touch ID, etc.)
#   - Missing/incorrect test artifacts (profiles that don't set the
#     expected keys, or are entirely missing)
#   - Fundamental state conflicts (shared profiles, user-scope
#     managed_policies that persist after removal)
#
# These are forced to MANUAL regardless of available scripts/profiles
# so the automated runner doesn't report spurious failures. Each entry
# should be accompanied by a comment describing why.
# Keyed by OS version.
NON_AUTOMATABLE_CIS_IDS: dict[str, dict[str, str]] = {
    "13": {},
    "14": {
        # 1.1: installing macOS updates inside a tart VM is unreliable
        # and slow, and the `softwareupdate -i -a` pass script can't
        # guarantee `software_update_required='0'` on an ephemeral VM.
        "1.1": "Requires real hardware with a connected Apple ID to install updates",
        # 2.1.1.1: only an enable profile exists; no disable variant
        # to flip the state, so ORG_DECISION testing is incomplete.
        "2.1.1.1": "Missing 2.1.1.1-disable.mobileconfig for ORG_DECISION testing",
        # 2.1.1.2: ORG_DECISION both directions fail because the
        # managed_policies user-scope values persist after profile
        # removal, corrupting subsequent push/verify cycles.
        "2.1.1.2": "User-scope managed_policies persist after profile removal",
        # 2.5.1 main + 3 field pairs: the Siri profiles (enable/disable)
        # are SHARED across four tests (main Siri plus three field
        # pairs), and the user-scope `allowAssistant` doesn't cleanly
        # flip between pushes, causing unpredictable cross-test state.
        "2.5.1": "Shared Siri profile causes cross-test state pollution",
        # 2.6.1.1: Location Services can't be enabled via MDM alone —
        # macOS requires explicit user privacy consent that a VM
        # cannot provide.
        "2.6.1.1": "VM cannot satisfy user-privacy gate for Location Services",
        # 2.6.3: the existing 2.6.3.mobileconfig sets
        # allowApplePersonalizedAdvertising (wrong key — that's 2.6.4)
        # instead of SubmitDiagInfo.AutoSubmit.
        "2.6.3": "Existing profile sets wrong keys; needs a corrected mobileconfig",
        # 2.8.1: Universal Control profiles do not reliably toggle
        # the managed_policies values in a way the query can observe.
        "2.8.1": "Universal Control profile toggling unreliable",
    },
    "15": {},
    "26": {},
}

# ---------------------------------------------------------------------------
# fleetctl config reading
# ---------------------------------------------------------------------------


def read_fleetctl_config(context: str = "default") -> dict:
    """Read URL and token from the fleetctl config file (~/.fleet/config).

    Returns a dict with 'address' and 'token' keys (may be empty strings).
    """
    config_path = Path.home() / ".fleet" / "config"
    if not config_path.exists():
        return {"address": "", "token": ""}

    with open(config_path) as f:
        config = yaml.safe_load(f)

    if not config or "contexts" not in config:
        return {"address": "", "token": ""}

    ctx = config["contexts"].get(context, {})
    return {
        "address": (ctx.get("address") or "").rstrip("/"),
        "token": ctx.get("token") or "",
    }


# ---------------------------------------------------------------------------
# Data classes
# ---------------------------------------------------------------------------


@dataclass
class Policy:
    cis_id: str
    name: str
    query: str
    resolution: str
    tags: str
    needs_mdm: bool


@dataclass
class TestPlan:
    policy: Policy
    test_type: str  # PASS_FAIL, PASS_ONLY, PROFILE, ORG_DECISION, MANUAL
    pass_script: Path | None = None
    fail_script: Path | None = None
    pass_only_script: Path | None = None
    profiles: list[Path] = field(default_factory=list)
    # For ORG_DECISION: the counterpart policy and the enable/disable profiles
    counterpart: Policy | None = None
    enable_profile: Path | None = None
    disable_profile: Path | None = None


@dataclass
class TestResult:
    cis_id: str
    name: str
    status: str  # PASS, FAIL, SKIP, ERROR
    details: str = ""


@dataclass
class FleetTeam:
    name: str
    team_id: int = 0
    enroll_secret: str = ""
    host_id: int | None = None
    hostname: str | None = None


# ---------------------------------------------------------------------------
# Logging helpers
# ---------------------------------------------------------------------------

VERBOSE = False


def log(msg: str) -> None:
    print(f"[*] {msg}", flush=True)


def log_verbose(msg: str) -> None:
    if VERBOSE:
        print(f"    {msg}", flush=True)


def log_error(msg: str) -> None:
    print(f"[!] {msg}", file=sys.stderr, flush=True)


# ---------------------------------------------------------------------------
# Policy parsing
# ---------------------------------------------------------------------------


def parse_policies(yaml_path: Path) -> list[Policy]:
    """Parse the multi-document YAML policy file."""
    policies = []
    missing_cis_id: list[str] = []
    with open(yaml_path) as f:
        for doc in yaml.safe_load_all(f):
            if not doc or doc.get("kind") != "policy":
                continue
            spec = doc.get("spec", {})
            name = spec.get("name", "")
            if not name.startswith("CIS -"):
                continue
            query = spec.get("query", "").strip()
            cis_id = spec.get("cis_id", "") or ""
            if not cis_id:
                missing_cis_id.append(name)
                continue
            policies.append(
                Policy(
                    cis_id=cis_id,
                    name=name,
                    query=query,
                    resolution=spec.get("resolution", ""),
                    tags=spec.get("tags", ""),
                    needs_mdm="managed_policies" in query,
                )
            )
    if missing_cis_id:
        # Skip policies without cis_id — without it, the runner can't
        # map to scripts/profiles and would generate paths like
        # CIS__pass.sh. Warn so the author can fix the YAML.
        for n in missing_cis_id:
            log_error(f"Policy has no cis_id, skipping: {n}")
    return policies


def filter_policies(
    policies: list[Policy],
    cis_ids: list[str] | None = None,
    match_terms: list[str] | None = None,
) -> list[Policy]:
    """Filter policies by CIS ID list or name substring match."""
    if cis_ids:
        id_set = set(cis_ids)
        # Handle combined IDs like "5.2.3, 5.2.4"
        result = []
        for p in policies:
            policy_ids = {s.strip() for s in p.cis_id.split(",")}
            if policy_ids & id_set:
                result.append(p)
        return result
    if match_terms:
        result = []
        for p in policies:
            if any(term.lower() in p.name.lower() for term in match_terms):
                result.append(p)
        return result
    return policies  # --all


def cis_id_sort_key(cis_id: str) -> list[int]:
    """Sort key for CIS IDs like '2.3.3.4' -> [2, 3, 3, 4]."""
    primary = cis_id.split(",")[0].strip()
    parts = []
    for part in primary.split("."):
        try:
            parts.append(int(part))
        except ValueError:
            parts.append(999)
    return parts


def _discover_profiles(cis_id: str, policy_cis_id: str, profiles_dir: Path) -> list[Path]:
    """Find all profiles for a CIS ID."""
    if not profiles_dir.exists():
        return []

    all_ids = [s.strip() for s in policy_cis_id.split(",")]
    candidates = [
        f"{cis_id}.mobileconfig",
        f"{cis_id}-enable.mobileconfig",
        f"{cis_id}-disable.mobileconfig",
        f"{cis_id}.enable.mobileconfig",
        f"{cis_id}.disable.mobileconfig",
    ]
    if len(all_ids) > 1:
        combined = "-and-".join(all_ids)
        candidates.append(f"{combined}.mobileconfig")

    profiles = []
    for pattern in candidates:
        p = profiles_dir / pattern
        if p.exists() and not p.name.startswith("not_"):
            profiles.append(p)
    for p in sorted(profiles_dir.glob(f"{cis_id}-part*.mobileconfig")):
        if not p.name.startswith("not_"):
            profiles.append(p)
    return profiles


def _find_enable_disable_profiles(cis_id: str, profiles_dir: Path) -> tuple[Path | None, Path | None]:
    """Find the enable and disable profile variants for an org-decision CIS ID."""
    enable = None
    disable = None
    for pattern_en, pattern_dis in [
        (f"{cis_id}-enable.mobileconfig", f"{cis_id}-disable.mobileconfig"),
        (f"{cis_id}.enable.mobileconfig", f"{cis_id}.disable.mobileconfig"),
    ]:
        p_en = profiles_dir / pattern_en
        p_dis = profiles_dir / pattern_dis
        if p_en.exists():
            enable = p_en
        if p_dis.exists():
            disable = p_dis
    return enable, disable


def _policy_stem(name: str) -> str:
    """Normalize a policy name to a stem for pairing org-decision variants."""
    s = name.lower()
    # Strip in order from most specific to least, to avoid partial matches
    for phrase in [
        "is enabled", "is disabled",
        "is true", "is false",
        "enabled", "disabled",
        "enable", "disable",
        "true", "false",
    ]:
        s = s.replace(phrase, "")
    # Collapse whitespace
    return " ".join(s.split())


def _is_enable_variant(policy: Policy) -> bool:
    """Heuristic: does this policy name indicate the 'enabled' variant?"""
    name_lower = policy.name.lower()
    # Check for explicit "enabled"/"is enabled"/"is true" patterns
    # but not "is disabled" which also contains "enable"
    if "disabled" in name_lower or "is false" in name_lower:
        return False
    if "enabled" in name_lower or "is true" in name_lower:
        return True
    # Default: treat as enable variant
    return True


def build_test_plans(
    policies: list[Policy],
    scripts_dir: Path,
    profiles_dir: Path,
    ssh_breaking_ids: set[str] | None = None,
    password_policy_ids: set[str] | None = None,
    non_automatable_ids: dict[str, str] | None = None,
) -> list[TestPlan]:
    """Map each policy to its test artifacts.

    ssh_breaking_ids: CIS IDs whose scripts disable SSH.
    password_policy_ids: CIS IDs whose MDM profiles break VM password
    authentication.
    non_automatable_ids: CIS ID -> reason. These are forced to MANUAL
    because the test cannot run reliably (VM limits, missing profiles,
    state conflicts). All three must be version-specific since CIS
    section numbers are not stable.
    """
    if ssh_breaking_ids is None:
        ssh_breaking_ids = set()
    if password_policy_ids is None:
        password_policy_ids = set()
    if non_automatable_ids is None:
        non_automatable_ids = {}

    # First pass: group policies by CIS ID to detect org-decision pairs
    from collections import defaultdict
    by_cis_id: dict[str, list[Policy]] = defaultdict(list)
    for p in policies:
        by_cis_id[p.cis_id].append(p)

    # Pair up org-decision policies. Two policies form a pair if they
    # share a CIS ID and one is the enable/true variant while the other
    # is the disable/false variant of the same setting.
    # Returns a dict mapping policy name -> (enable_pol, disable_pol)
    org_decision_pairs: dict[str, tuple[Policy, Policy]] = {}
    org_decision_policies: set[str] = set()  # names of policies in a pair

    for cis_id, pols in by_cis_id.items():
        if len(pols) < 2:
            continue
        enables = [p for p in pols if _is_enable_variant(p)]
        disables = [p for p in pols if not _is_enable_variant(p)]
        # Match pairs by finding the shared "stem" in the name. E.g.:
        #   "Siri field TypeToSiriEnabled is true" pairs with
        #   "Siri field TypeToSiriEnabled is false"
        paired_disables = set()
        for en_pol in enables:
            # Normalize: strip all enable/disable/true/false variants
            # to find the common stem between paired policies
            en_stem = _policy_stem(en_pol.name)
            best_match = None
            for dis_pol in disables:
                if dis_pol.name in paired_disables:
                    continue
                dis_stem = _policy_stem(dis_pol.name)
                if en_stem == dis_stem:
                    best_match = dis_pol
                    break
            if best_match:
                paired_disables.add(best_match.name)
                org_decision_pairs[en_pol.name] = (en_pol, best_match)
                org_decision_policies.add(en_pol.name)
                org_decision_policies.add(best_match.name)

    plans = []
    for policy in sorted(policies, key=lambda p: cis_id_sort_key(p.cis_id)):
        cis_id = policy.cis_id.split(",")[0].strip()

        pass_script = scripts_dir / f"CIS_{cis_id}_pass.sh"
        fail_script = scripts_dir / f"CIS_{cis_id}_fail.sh"
        pass_only_script = scripts_dir / f"CIS_{cis_id}.sh"

        # Check for not_always_working variants
        naw_script = scripts_dir / f"not_always_working_CIS_{cis_id}.sh"
        if naw_script.exists() and not pass_only_script.exists():
            log_verbose(
                f"Skipping not_always_working script for {cis_id}"
            )

        profiles = _discover_profiles(cis_id, policy.cis_id, profiles_dir)

        # Org-decision pairs: create one ORG_DECISION plan per pair.
        # Skip the disable variant — it's covered by the enable's plan.
        if policy.name in org_decision_policies:
            if policy.name not in org_decision_pairs:
                continue  # This is the disable variant, skip it

            enable_pol, disable_pol = org_decision_pairs[policy.name]
            enable_prof, disable_prof = _find_enable_disable_profiles(cis_id, profiles_dir)

            # Non-automatable org-decision pairs fall back to MANUAL.
            if cis_id in non_automatable_ids:
                test_type = "MANUAL"
                log_verbose(
                    f"CIS {cis_id}: forced to MANUAL "
                    f"({non_automatable_ids[cis_id]})"
                )
            elif enable_prof or disable_prof:
                test_type = "ORG_DECISION"
            else:
                test_type = "MANUAL"

            plans.append(
                TestPlan(
                    policy=enable_pol,
                    test_type=test_type,
                    profiles=profiles,
                    counterpart=disable_pol,
                    enable_profile=enable_prof,
                    disable_profile=disable_prof,
                )
            )
            continue

        # Tests that disable SSH or break password auth would lock us
        # out of the VM. Force these to MANUAL so the user is prompted
        # instead.
        if cis_id in ssh_breaking_ids:
            test_type = "MANUAL"
            log_verbose(
                f"CIS {cis_id}: forced to MANUAL (script disables SSH)"
            )
        elif cis_id in password_policy_ids:
            test_type = "MANUAL"
            log_verbose(
                f"CIS {cis_id}: forced to MANUAL "
                "(password policy profile breaks VM SSH auth)"
            )
        elif cis_id in non_automatable_ids:
            test_type = "MANUAL"
            log_verbose(
                f"CIS {cis_id}: forced to MANUAL "
                f"({non_automatable_ids[cis_id]})"
            )
        elif pass_script.exists() and fail_script.exists():
            test_type = "PASS_FAIL"
        elif pass_only_script.exists():
            test_type = "PASS_ONLY"
        elif profiles:
            test_type = "PROFILE"
        else:
            test_type = "MANUAL"

        plans.append(
            TestPlan(
                policy=policy,
                test_type=test_type,
                pass_script=pass_script if pass_script.exists() else None,
                fail_script=fail_script if fail_script.exists() else None,
                pass_only_script=(
                    pass_only_script if pass_only_script.exists() else None
                ),
                profiles=profiles,
            )
        )
    return plans


# ---------------------------------------------------------------------------
# Shell / subprocess helpers
# ---------------------------------------------------------------------------


def run_cmd(
    cmd: list[str], timeout: int = 120, check: bool = True, capture: bool = True
) -> subprocess.CompletedProcess:
    """Run a command, return CompletedProcess."""
    log_verbose(f"$ {' '.join(cmd)}")
    result = subprocess.run(
        cmd,
        capture_output=capture,
        text=True,
        timeout=timeout,
    )
    if capture and result.stdout:
        log_verbose(result.stdout.rstrip())
    if capture and result.stderr:
        log_verbose(result.stderr.rstrip())
    if check and result.returncode != 0:
        # Surface the error output before raising
        stderr_msg = (result.stderr or "").strip()
        stdout_msg = (result.stdout or "").strip()
        error_detail = stderr_msg or stdout_msg or "(no output)"
        log_error(f"Command failed: {' '.join(cmd)}")
        log_error(f"  {error_detail}")
        raise subprocess.CalledProcessError(
            result.returncode, cmd, result.stdout, result.stderr
        )
    return result


def ssh(ip: str, command: str, timeout: int = 120) -> subprocess.CompletedProcess:
    """Run a command on the VM via SSH."""
    cmd = [
        "sshpass", "-p", VM_PASS,
        "ssh",
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=/dev/null",
        "-o", "LogLevel=ERROR",
        f"{VM_USER}@{ip}",
        command,
    ]
    return run_cmd(cmd, timeout=timeout, check=False)


def scp_to_vm(ip: str, local_path: Path, remote_path: str) -> None:
    """Copy a file to the VM."""
    cmd = [
        "sshpass", "-p", VM_PASS,
        "scp",
        "-o", "StrictHostKeyChecking=no",
        "-o", "UserKnownHostsFile=/dev/null",
        "-o", "LogLevel=ERROR",
        str(local_path),
        f"{VM_USER}@{ip}:{remote_path}",
    ]
    run_cmd(cmd)


# ---------------------------------------------------------------------------
# Fleet API helpers
# ---------------------------------------------------------------------------


def fleet_api(
    fleet_url: str,
    token: str,
    method: str,
    path: str,
    body: dict | None = None,
) -> dict | None:
    """Make a Fleet API request. Returns parsed JSON or None."""
    url = f"{fleet_url}{path}"
    data = json.dumps(body).encode() if body else None
    req = Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {token}")
    if data:
        req.add_header("Content-Type", "application/json")
    try:
        with urlopen(req, timeout=30) as resp:
            raw = resp.read()
            if raw:
                return json.loads(raw)
            return None
    except HTTPError as e:
        body_text = e.read().decode() if e.fp else ""
        log_error(f"Fleet API {method} {path} returned {e.code}: {body_text}")
        raise
    except URLError as e:
        log_error(f"Fleet API {method} {path} failed: {e.reason}")
        raise


# ---------------------------------------------------------------------------
# Fleet team management
# ---------------------------------------------------------------------------


def create_fleet_team(
    fleet_url: str, token: str, fleetctl: str, team_name: str, enroll_secret: str
) -> FleetTeam:
    """Create a Fleet team via fleetctl apply and return the team info."""
    # Note: "kind: fleet" / "spec.fleet" is the new naming but silently
    # fails to create teams as of fleetctl v4.x. Use the deprecated
    # "kind: team" / "spec.team" which actually works.
    team_yaml = {
        "apiVersion": "v1",
        "kind": "team",
        "spec": {
            "team": {
                "name": team_name,
                "secrets": [{"secret": enroll_secret}],
            }
        },
    }

    TMP_DIR.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".yml", delete=False, dir=str(TMP_DIR)
    ) as f:
        yaml.dump(team_yaml, f)
        tmp_path = f.name

    try:
        log(f"Creating Fleet team: {team_name}")
        run_cmd([fleetctl, "apply", "-f", tmp_path])
    finally:
        os.unlink(tmp_path)

    # Get team ID. Try --name filter first, fall back to scanning all.
    team_id = 0
    for get_cmd in [
        [fleetctl, "get", "teams", "--json", "--name", team_name],
        [fleetctl, "get", "teams", "--json"],
    ]:
        result = run_cmd(get_cmd, check=False)
        if result.returncode != 0 or not result.stdout:
            continue
        for line in result.stdout.strip().splitlines():
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                team_data = json.loads(line)
            except json.JSONDecodeError:
                continue
            spec = team_data.get("spec", {})
            team_obj = spec.get("fleet") or spec.get("team") or {}
            if team_obj.get("name") == team_name:
                team_id = team_obj.get("id", 0)
                break
        if team_id:
            break

    if team_id == 0:
        log_error("Could not determine team ID after creation")

    log(f"Team created: {team_name} (ID: {team_id})")
    return FleetTeam(name=team_name, team_id=team_id, enroll_secret=enroll_secret)


def push_profiles_to_team(
    fleet_url: str,
    token: str,
    fleetctl: str,
    team_name: str,
    profiles: list[Path],
) -> None:
    """Update team MDM settings to push mobileconfig profiles.

    Deduplicates by path so callers can include the same profile
    multiple times (e.g., an org-decision profile used by several
    pairs) without triggering PayloadDisplayName conflicts in Fleet.
    """
    # Dedupe by absolute path
    seen: set[str] = set()
    unique: list[Path] = []
    for p in profiles:
        key = str(p.resolve()) if hasattr(p, "resolve") else str(p)
        if key in seen:
            continue
        seen.add(key)
        unique.append(p)
    if not unique:
        # Push an empty list to clear the team's profiles
        custom_settings = []
    else:
        custom_settings = [{"path": str(p)} for p in unique]

    team_yaml = {
        "apiVersion": "v1",
        "kind": "team",
        "spec": {
            "team": {
                "name": team_name,
                "mdm": {
                    "macos_settings": {
                        "custom_settings": custom_settings,
                    }
                },
            }
        },
    }

    TMP_DIR.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".yml", delete=False, dir=str(TMP_DIR)
    ) as f:
        yaml.dump(team_yaml, f)
        tmp_path = f.name

    try:
        log(f"Pushing {len(unique)} MDM profile(s) to team {team_name}")
        run_cmd([fleetctl, "apply", "-f", tmp_path])
    finally:
        os.unlink(tmp_path)


def delete_fleet_team(fleet_url: str, token: str, team_id: int) -> None:
    """Delete a Fleet team and all its associated data."""
    log(f"Deleting Fleet team ID {team_id}")
    try:
        fleet_api(fleet_url, token, "DELETE", f"/api/v1/fleet/teams/{team_id}")
    except HTTPError:
        log_error(f"Failed to delete team {team_id}")


def delete_fleet_host(fleet_url: str, token: str, host_id: int) -> None:
    """Delete a host from Fleet."""
    log(f"Deleting Fleet host ID {host_id}")
    try:
        fleet_api(fleet_url, token, "DELETE", f"/api/v1/fleet/hosts/{host_id}")
    except HTTPError:
        log_error(f"Failed to delete host {host_id}")


def transfer_host_to_team(
    fleet_url: str, token: str, host_id: int, team_id: int
) -> None:
    """Move a host into a specific team."""
    log(f"Transferring host {host_id} to team {team_id}")
    try:
        fleet_api(
            fleet_url, token, "POST",
            "/api/v1/fleet/hosts/transfer",
            body={"team_id": team_id, "hosts": [host_id]},
        )
    except HTTPError as e:
        log_error(f"Failed to transfer host {host_id} to team {team_id}: {e}")
        raise


def get_host_by_hostname(
    fleet_url: str, token: str, hostname: str
) -> dict | None:
    """Look up a host by hostname in Fleet (case-insensitive)."""
    from urllib.parse import quote

    target = hostname.lower()
    data = fleet_api(
        fleet_url, token, "GET", f"/api/v1/fleet/hosts?query={quote(hostname)}"
    )
    if data and data.get("hosts"):
        for host in data["hosts"]:
            if (host.get("hostname") or "").lower() == target:
                return host
    return None


# ---------------------------------------------------------------------------
# Fleet agent build
# ---------------------------------------------------------------------------


def build_fleet_pkg(fleet_url: str, enroll_secret: str, fleetctl: str) -> Path:
    """Build a fleet-osquery.pkg using fleetctl."""
    pkg_path = Path("fleet-osquery.pkg")
    if pkg_path.exists():
        pkg_path.unlink()

    log("Building fleet agent package...")
    run_cmd([
        fleetctl, "package",
        "--type=pkg",
        "--enable-scripts",
        "--fleet-desktop",
        "--disable-open-folder",
        f"--fleet-url={fleet_url}",
        f"--enroll-secret={enroll_secret}",
    ], timeout=300)

    if not pkg_path.exists():
        raise RuntimeError("fleetctl package did not produce fleet-osquery.pkg")

    log(f"Package built: {pkg_path}")
    return pkg_path


# ---------------------------------------------------------------------------
# VM management (tart)
# ---------------------------------------------------------------------------


def vm_exists(name: str) -> bool:
    result = run_cmd(["tart", "list"], check=False)
    return name in (result.stdout or "")


def vm_is_running(name: str) -> bool:
    """Check if a tart VM is currently running (has an IP)."""
    if not vm_exists(name):
        return False
    result = run_cmd(["tart", "ip", name], check=False, timeout=10)
    return result.returncode == 0 and bool(result.stdout.strip())


def create_vm(name: str, image: str) -> None:
    if vm_exists(name):
        log(f"VM {name} already exists, deleting...")
        run_cmd(["tart", "stop", name], check=False)
        time.sleep(2)
        run_cmd(["tart", "delete", name], check=False)

    log(f"Cloning VM {name} from {image}...")
    run_cmd(["tart", "clone", image, name], timeout=600)


def start_vm(name: str) -> subprocess.Popen:
    log(f"Starting VM {name}...")
    proc = subprocess.Popen(
        ["tart", "run", name],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return proc


def wait_for_ip(name: str, timeout: int = 180) -> str:
    """Poll for the VM's IP address."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        result = run_cmd(["tart", "ip", name], check=False, timeout=10)
        if result.returncode == 0 and result.stdout.strip():
            ip = result.stdout.strip()
            log(f"VM IP: {ip}")
            return ip
        time.sleep(3)
    raise TimeoutError(f"VM {name} did not get an IP within {timeout}s")


def wait_for_ssh(ip: str, timeout: int = 120) -> None:
    """Wait until SSH is available on the VM."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        result = ssh(ip, "echo ok", timeout=10)
        if result.returncode == 0:
            log("SSH is ready")
            return
        time.sleep(3)
    raise TimeoutError(f"SSH not available on {ip} within {timeout}s")


def stop_vm(name: str) -> None:
    run_cmd(["tart", "stop", name], check=False)


def delete_vm(name: str) -> None:
    run_cmd(["tart", "stop", name], check=False)
    time.sleep(2)
    run_cmd(["tart", "delete", name], check=False)


# ---------------------------------------------------------------------------
# Agent installation and enrollment
# ---------------------------------------------------------------------------


def install_agent(ip: str, pkg_path: Path) -> None:
    """Copy and install the fleet agent on the VM."""
    log("Copying agent package to VM...")
    scp_to_vm(ip, pkg_path, "fleet-osquery.pkg")

    log("Installing agent...")
    result = ssh(
        ip,
        f"echo {VM_PASS} | sudo -S installer -pkg fleet-osquery.pkg -target /",
        timeout=120,
    )
    if result.returncode != 0:
        raise RuntimeError(f"Agent install failed: {result.stderr}")


def wait_for_identifier(ip: str, timeout: int = 120) -> str:
    """Wait for /opt/orbit/identifier to appear and return its content."""
    log("Waiting for orbit identifier...")
    deadline = time.time() + timeout
    while time.time() < deadline:
        result = ssh(ip, "cat /opt/orbit/identifier 2>/dev/null", timeout=10)
        if result.returncode == 0 and result.stdout.strip():
            identifier = result.stdout.strip()
            log(f"Orbit identifier: {identifier}")
            return identifier
        time.sleep(5)
    raise TimeoutError("Orbit identifier did not appear")


def wait_for_fleet_registration(
    fleet_url: str, identifier: str, timeout: int = 120
) -> None:
    """Poll until the host is registered in Fleet."""
    log("Waiting for Fleet registration...")
    deadline = time.time() + timeout
    url = f"{fleet_url}/device/{identifier}"
    while time.time() < deadline:
        try:
            req = Request(url)
            with urlopen(req, timeout=10) as resp:
                if resp.status == 200:
                    log("Host registered in Fleet")
                    return
        except (HTTPError, URLError):
            pass
        time.sleep(5)
    raise TimeoutError("Host did not register in Fleet")


def get_hostname(ip: str) -> str:
    """Get the VM's hostname via SSH."""
    result = ssh(ip, "hostname", timeout=10)
    if result.returncode != 0:
        raise RuntimeError("Could not get hostname from VM")
    return result.stdout.strip()


def enroll_mdm(ip: str, fleet_url: str, identifier: str) -> None:
    """Download and open the MDM enrollment profile, prompt user.

    Fetches the enrollment profile from the Fleet device API. If the
    Fleet server has Apple MDM configured, this returns an actual
    .mobileconfig XML plist. If not, it returns a JSON with an
    enroll_url — in that case we fetch the profile from that URL.
    """
    log("Fetching MDM enrollment profile...")
    mdm_api_url = (
        f"{fleet_url}/api/latest/fleet/device/"
        f"{identifier}/mdm/apple/manual_enrollment_profile"
    )

    # Download on the host side first to check what we got
    try:
        req = Request(mdm_api_url)
        with urlopen(req, timeout=30) as resp:
            content = resp.read()
    except (HTTPError, URLError) as e:
        raise RuntimeError(f"Failed to fetch MDM enrollment profile: {e}")

    # The device API may return:
    # a) An actual mobileconfig (legacy)
    # b) A JSON with enroll_url pointing to /enroll?enroll_secret=...
    #    The actual profile is at /api/v1/fleet/enrollment_profiles/ota
    #    with the same enroll_secret query parameter.
    if b"<?xml" not in content and b"plist" not in content:
        try:
            data = json.loads(content)
            enroll_url = data.get("enroll_url", "")
        except (json.JSONDecodeError, ValueError):
            enroll_url = ""

        if enroll_url:
            # Extract enroll_secret from the URL and build OTA URL
            from urllib.parse import urlparse, parse_qs
            parsed = urlparse(enroll_url)
            qs = parse_qs(parsed.query)
            enroll_secret = qs.get("enroll_secret", [""])[0]
            if not enroll_secret:
                raise RuntimeError(
                    f"Could not extract enroll_secret from: {enroll_url}"
                )

            ota_url = (
                f"{fleet_url}/api/v1/fleet/enrollment_profiles/ota"
                f"?enroll_secret={enroll_secret}"
            )
            log_verbose(f"Fetching OTA enrollment profile from {ota_url}")
            try:
                req = Request(ota_url)
                with urlopen(req, timeout=30) as resp:
                    content = resp.read()
            except (HTTPError, URLError) as e:
                raise RuntimeError(
                    f"Failed to fetch OTA enrollment profile: {e}. "
                    "Ensure Apple MDM is configured on the Fleet server."
                )

    if b"<?xml" not in content and b"plist" not in content:
        raise RuntimeError(
            "Fleet returned an invalid MDM enrollment profile. "
            "Ensure Apple MDM is fully configured on the Fleet server "
            "(APNs certificate, SCEP). Without MDM, profile-based "
            "CIS policies cannot be tested."
        )

    # Write to temp file, SCP to VM, open
    TMP_DIR.mkdir(parents=True, exist_ok=True)
    tmp_profile = TMP_DIR / "mdm_profile.mobileconfig"
    tmp_profile.write_bytes(content)
    scp_to_vm(ip, tmp_profile, "mdm_profile.mobileconfig")
    tmp_profile.unlink(missing_ok=True)

    ssh(ip, "open mdm_profile.mobileconfig", timeout=10)
    time.sleep(1)
    ssh(
        ip,
        "open x-apple.systempreferences:com.apple.preferences.configurationprofiles",
        timeout=10,
    )

    print()
    print("=" * 60)
    print("MDM ENROLLMENT REQUIRED")
    print("=" * 60)
    print()
    print("The MDM enrollment profile has been opened on the VM.")
    print("Complete the following steps in the tart VM GUI:")
    print("  1. Go to System Settings > Privacy & Security > Profiles")
    print("  2. Click on the Fleet MDM profile")
    print("  3. Click 'Enroll' and enter the password: admin")
    print()
    input("Press Enter when MDM enrollment is complete...")
    print()


# ---------------------------------------------------------------------------
# Query execution
# ---------------------------------------------------------------------------


def wait_for_query_pass(
    query: str,
    hostname: str,
    fleetctl: str,
    query_timeout: int,
    deadline_seconds: int = 90,
    poll_interval: int = 10,
    expected: bool = True,
) -> bool:
    """Poll run_query until it matches `expected` or deadline elapses.

    expected=True  (default): wait for the query to return rows (e.g.,
                              after pushing a profile that should make
                              the policy pass).
    expected=False:           wait for the query to return no rows
                              (e.g., after removing a profile that was
                              the reason the policy passed).

    Returns True when the expected state is observed, False on timeout.
    """
    deadline = time.time() + deadline_seconds
    while time.time() < deadline:
        if run_query(query, hostname, fleetctl, query_timeout) == expected:
            return True
        time.sleep(poll_interval)
    return False


def run_query(
    query: str, hostname: str, fleetctl: str, timeout: int = 60
) -> bool:
    """Run a policy query via fleetctl. Returns True if query returns rows."""
    result = run_cmd(
        [
            fleetctl, "query",
            f"--query={query}",
            f"--hosts={hostname}",
            "--exit",
            f"--timeout={timeout}s",
        ],
        timeout=timeout + 30,
        check=False,
    )

    if result.returncode != 0:
        detail = (result.stderr or result.stdout or "").strip() or "(no output)"
        raise RuntimeError(f"fleetctl query failed: {detail}")

    stdout = result.stdout or ""
    # fleetctl query outputs one JSON object per line per host:
    #   {"host":"hostname","rows":[{"1":"1",...}]}
    # If the query matches, "rows" is a non-empty array.
    # If it doesn't match, "rows" is an empty array [].
    # There may also be status lines like "100% responded..."
    for line in stdout.strip().splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            data = json.loads(line)
        except json.JSONDecodeError:
            continue
        rows = data.get("rows", [])
        if rows:
            log_verbose(f"Query returned {len(rows)} row(s)")
            return True

    log_verbose("Query returned no rows")
    return False


# ---------------------------------------------------------------------------
# Script execution on VM
# ---------------------------------------------------------------------------


def run_script_on_vm(ip: str, script_path: Path) -> subprocess.CompletedProcess:
    """Copy a script to the VM and execute it.

    Raises RuntimeError on non-zero exit so the caller surfaces the
    failure as ERROR (with the script's stderr) rather than masking it
    as a spurious PASS/FAIL on the follow-up query.
    """
    remote_name = f"/tmp/{script_path.name}"
    scp_to_vm(ip, script_path, remote_name)
    ssh(ip, f"chmod +x {remote_name}", timeout=10)
    result = ssh(
        ip,
        f"echo {VM_PASS} | sudo -S {remote_name}",
        timeout=300,
    )
    log_verbose(f"Script exit code: {result.returncode}")
    if result.returncode != 0:
        detail = (result.stderr or result.stdout or "").strip() or "(no output)"
        raise RuntimeError(
            f"Script {script_path.name} failed (exit {result.returncode}): {detail}"
        )
    return result


# ---------------------------------------------------------------------------
# Test execution
# ---------------------------------------------------------------------------


def prompt_manual(policy: Policy) -> bool:
    """Prompt user for manual remediation. Returns True if applied, False if skipped."""
    print()
    print("-" * 60)
    print(f"MANUAL: CIS {policy.cis_id} - {policy.name}")
    print("-" * 60)
    print()
    print("Resolution:")
    for line in policy.resolution.strip().splitlines():
        print(f"  {line}")
    print()
    response = input("Apply the remediation above in the VM, then press Enter (or type 'skip'): ")
    return response.strip().lower() != "skip"


def run_test(
    plan: TestPlan,
    ip: str,
    hostname: str,
    fleetctl: str,
    query_timeout: int,
    pre_profile_passed: set[str] | None = None,
    fleet_url: str = "",
    fleet_token: str = "",
    team_name: str = "",
    base_profiles: list[Path] | None = None,
) -> TestResult:
    """Execute a single test plan and return the result."""
    policy = plan.policy
    cis_id = policy.cis_id
    name = policy.name

    log(f"Testing CIS {cis_id}: {name} [{plan.test_type}]")

    try:
        if plan.test_type == "PASS_FAIL":
            # For MDM-backed policies, the local fail/pass scripts
            # can't remove MDM-delivered profiles. Instead, toggle
            # the profile on/off via the team's MDM settings.
            uses_mdm = policy.needs_mdm and plan.profiles
            if uses_mdm:
                other_profiles = list(base_profiles or [])
                # Drop this policy's profiles from the base set
                this_profile_paths = {str(p) for p in plan.profiles}
                other_profiles = [
                    p for p in other_profiles if str(p) not in this_profile_paths
                ]

                mdm_deadline = 90

                # Step 1: remove profile → query should poll to False
                log_verbose("Removing profile via MDM...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name, other_profiles
                )
                if not wait_for_query_pass(
                    policy.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=mdm_deadline, expected=False,
                ):
                    # Restore before returning
                    push_profiles_to_team(
                        fleet_url, fleet_token, fleetctl, team_name,
                        other_profiles + list(plan.profiles),
                    )
                    return TestResult(
                        cis_id, name, "FAIL",
                        "Query still passed after removing MDM profile "
                        f"(waited {mdm_deadline}s)",
                    )

                # Step 2: re-install profile → query should poll to True
                log_verbose("Re-pushing profile via MDM...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name,
                    other_profiles + list(plan.profiles),
                )
                if not wait_for_query_pass(
                    policy.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=mdm_deadline, expected=True,
                ):
                    return TestResult(
                        cis_id, name, "FAIL",
                        "Query did not pass after re-pushing MDM profile "
                        f"(waited {mdm_deadline}s)",
                    )
                return TestResult(cis_id, name, "PASS")

            # Non-MDM PASS_FAIL: run the local scripts
            log_verbose("Running fail script...")
            run_script_on_vm(ip, plan.fail_script)
            time.sleep(5)
            if run_query(policy.query, hostname, fleetctl, query_timeout):
                return TestResult(
                    cis_id, name, "FAIL",
                    "Expected query to fail after fail script, but it returned rows",
                )

            log_verbose("Running pass script...")
            run_script_on_vm(ip, plan.pass_script)
            time.sleep(5)
            if not run_query(policy.query, hostname, fleetctl, query_timeout):
                return TestResult(
                    cis_id, name, "FAIL",
                    "Expected query to pass after pass script, but it returned no rows",
                )

            return TestResult(cis_id, name, "PASS")

        elif plan.test_type == "PASS_ONLY":
            log_verbose("Running pass-only script...")
            run_script_on_vm(ip, plan.pass_only_script)
            time.sleep(5)
            if not run_query(policy.query, hostname, fleetctl, query_timeout):
                return TestResult(
                    cis_id, name, "FAIL",
                    "Query returned no rows after running pass script",
                )
            return TestResult(cis_id, name, "PASS")

        elif plan.test_type == "PROFILE":
            # Profile-only test. Profiles were pushed in Phase 5.
            # The post-profile query must pass. A pre-profile "passed"
            # state (captured in Phase 5 before profiles were pushed)
            # is noted as a warning in the details but does not fail
            # the test — some queries check OS state (firewall,
            # gatekeeper) that may be compliant regardless of profile.
            log_verbose(
                f"Profile(s) pushed: {[p.name for p in plan.profiles]}"
            )

            details = ""
            if pre_profile_passed and cis_id in pre_profile_passed:
                details = (
                    "note: query passed before profile delivery — "
                    "the OS state may satisfy this regardless of MDM"
                )

            # Poll instead of checking once — MDM delivery time varies
            # and the fixed 30s wait in Phase 5 isn't always enough.
            if not wait_for_query_pass(
                policy.query, hostname, fleetctl, query_timeout
            ):
                return TestResult(
                    cis_id, name, "FAIL",
                    "Query returned no rows after MDM profile delivery",
                )
            return TestResult(cis_id, name, "PASS", details)

        elif plan.test_type == "ORG_DECISION":
            # Org-decision pair: two contradicting policies share the
            # same CIS ID (e.g., iCloud Drive enabled vs disabled).
            # Test both directions using enable/disable profiles.
            enable_pol = policy
            disable_pol = plan.counterpart
            enable_prof = plan.enable_profile
            disable_prof = plan.disable_profile
            org_deadline = 90

            log_verbose(
                f"Org-decision pair: "
                f"enable={enable_prof.name if enable_prof else 'none'}, "
                f"disable={disable_prof.name if disable_prof else 'none'}"
            )

            failures = []

            # Include all base profiles alongside the org-decision
            # profile so we don't wipe them from the team.
            other_profiles = list(base_profiles or [])

            # Step 1: Push enable profile, poll: enable passes + disable fails
            if enable_prof:
                log_verbose("Pushing enable profile + base profiles...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name,
                    other_profiles + [enable_prof],
                )

                if not wait_for_query_pass(
                    enable_pol.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=org_deadline, expected=True,
                ):
                    failures.append(
                        "Enable policy did not pass after enable profile"
                    )
                if disable_pol and not wait_for_query_pass(
                    disable_pol.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=org_deadline, expected=False,
                ):
                    failures.append(
                        "Disable policy still passes after enable profile"
                    )

            # Step 2: Push disable profile, poll: disable passes + enable fails
            if disable_prof:
                log_verbose("Pushing disable profile + base profiles...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name,
                    other_profiles + [disable_prof],
                )

                if disable_pol and not wait_for_query_pass(
                    disable_pol.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=org_deadline, expected=True,
                ):
                    failures.append(
                        "Disable policy did not pass after disable profile"
                    )
                if not wait_for_query_pass(
                    enable_pol.query, hostname, fleetctl, query_timeout,
                    deadline_seconds=org_deadline, expected=False,
                ):
                    failures.append(
                        "Enable policy still passes after disable profile"
                    )

            # Restore base profiles without any org-decision profile.
            # No polling here — no assertion about the resulting state,
            # just letting the team settle before the next test.
            if other_profiles:
                log_verbose("Restoring base profiles...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name,
                    other_profiles,
                )
                time.sleep(15)

            if failures:
                return TestResult(
                    cis_id, name, "FAIL", "; ".join(failures),
                )
            return TestResult(cis_id, name, "PASS")

        elif plan.test_type == "MANUAL":
            applied = prompt_manual(policy)
            if not applied:
                return TestResult(cis_id, name, "SKIP", "User skipped")
            time.sleep(5)
            if not run_query(policy.query, hostname, fleetctl, query_timeout):
                return TestResult(
                    cis_id, name, "FAIL",
                    "Query returned no rows after manual remediation",
                )
            return TestResult(cis_id, name, "PASS")

    except subprocess.TimeoutExpired as e:
        return TestResult(cis_id, name, "ERROR", f"Timeout: {e}")
    except Exception as e:
        return TestResult(cis_id, name, "ERROR", str(e))

    return TestResult(cis_id, name, "ERROR", "Unexpected test type")


# ---------------------------------------------------------------------------
# Reporting
# ---------------------------------------------------------------------------


def print_summary(results: list[TestResult], team: FleetTeam) -> int:
    """Print test results summary. Returns exit code."""
    passed = [r for r in results if r.status == "PASS"]
    failed = [r for r in results if r.status == "FAIL"]
    skipped = [r for r in results if r.status == "SKIP"]
    errors = [r for r in results if r.status == "ERROR"]

    print()
    print("=" * 60)
    print("CIS Benchmark Test Results")
    print("=" * 60)
    print(f"Team: {team.name} (ID: {team.team_id})")
    if team.hostname:
        print(f"Host: {team.hostname}")
    print(
        f"Total: {len(results)}  "
        f"Pass: {len(passed)}  "
        f"Fail: {len(failed)}  "
        f"Skip: {len(skipped)}  "
        f"Error: {len(errors)}"
    )

    if failed:
        print()
        print("Failures:")
        for r in failed:
            print(f"  {r.cis_id}  {r.name}")
            print(f"       {r.details}")

    if errors:
        print()
        print("Errors:")
        for r in errors:
            print(f"  {r.cis_id}  {r.name}")
            print(f"       {r.details}")

    print()
    return 0 if not failed and not errors else 1


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="""\
Automated CIS benchmark policy test runner.

Creates a fresh macOS VM using tart, enrolls it in a dedicated Fleet
team, and runs each selected CIS policy query to verify that the
osquery SQL correctly detects compliance and non-compliance.

The tool creates a temporary Fleet team with its own enroll secret,
builds a fleet agent package bound to that team, and installs it in
the VM. If any selected policies require MDM profiles, the tool
prompts for interactive MDM enrollment and pushes the profiles to the
team automatically.

Each policy is tested according to the available test artifacts:
  - Scripts with _pass and _fail suffixes: verifies both directions
  - Scripts with just the CIS ID: verifies pass only
  - MDM profiles with no scripts: verifies fail without profile,
    then pass after profile delivery
  - No artifacts: prompts for manual remediation or skips

After all tests complete, a summary of pass/fail/skip/error is
printed. Exit code is 0 if all non-skipped tests passed, 1 otherwise.

Prerequisites:
  - tart (brew install cirruslabs/cli/tart)
  - sshpass (brew install esolitos/ipa/sshpass)
  - pyyaml (pip3 install pyyaml)
  - A running Fleet instance with MDM configured
  - A Fleet API token (Fleet UI > My account, or fleetctl login)
  - fleetctl built locally (make build from repo root)

Examples:
  # Test a single benchmark
  python3 %(prog)s --macos-version 14 --cis-ids 2.3.3.4

  # Test all benchmarks, skip those without scripts or profiles
  python3 %(prog)s --macos-version 14 --all --skip-manual

  # Test by name, clean up everything after
  python3 %(prog)s --macos-version 14 --match "Remote Login" --cleanup
""",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )

    # --- Target selection ---
    target = parser.add_argument_group(
        "target",
        "Which macOS version and CIS policies to test. Exactly one of "
        "--all, --cis-ids, or --match is required.",
    )
    target.add_argument(
        "--macos-version",
        required=True,
        choices=list(VERSION_MAP.keys()),
        help=(
            "macOS version to test. Determines the tart VM image and "
            "the CIS policy directory (ee/cis/macos-{version}/)."
        ),
    )

    selection = target.add_mutually_exclusive_group(required=True)
    selection.add_argument(
        "--all",
        action="store_true",
        help="Test every CIS policy in the policy file.",
    )
    selection.add_argument(
        "--cis-ids",
        metavar="IDS",
        help=(
            "Comma-separated list of CIS IDs to test "
            "(e.g., '1.1,2.3.3.4,5.2.3'). Matches against the cis_id "
            "field in the policy YAML."
        ),
    )
    selection.add_argument(
        "--match",
        metavar="TERMS",
        help=(
            "Comma-separated substrings to match against policy names "
            "(e.g., 'Remote Login,AirDrop'). Case-insensitive."
        ),
    )
    target.add_argument(
        "--skip-manual",
        "--skip-no-script",
        dest="skip_manual",
        action="store_true",
        help=(
            "Skip MANUAL-type tests (policies with no automation) "
            "instead of prompting the user to perform the steps. "
            "MDM profile tests and org-decision tests still run. "
            "(--skip-no-script is kept as a deprecated alias.)"
        ),
    )

    type_filter = target.add_mutually_exclusive_group()
    type_filter.add_argument(
        "--only-scripts",
        action="store_true",
        help=(
            "Only run script-based tests (PASS_FAIL and PASS_ONLY). "
            "Skip MDM profile tests, org-decision tests, and manual "
            "prompts. Useful for running a fast, fully-automated subset."
        ),
    )
    type_filter.add_argument(
        "--only-mdm",
        action="store_true",
        help=(
            "Only run MDM-dependent tests (PROFILE and ORG_DECISION). "
            "Skip script-based and manual tests. Useful when validating "
            "MDM profile delivery."
        ),
    )
    type_filter.add_argument(
        "--only-manual",
        action="store_true",
        help=(
            "Only run tests that require manual user interaction "
            "(MANUAL). Skip all automated tests. Useful for walking "
            "through the checks that can't be scripted."
        ),
    )

    # --- Fleet connection ---
    fleet = parser.add_argument_group(
        "fleet connection",
        "How to connect to the Fleet server. The tool creates a "
        "temporary team and builds a fleet agent bound to it. "
        "By default, the URL and token are read from the fleetctl "
        "config file (~/.fleet/config) so that a logged-in fleetctl "
        "session works with no extra flags. Use --fleet-url and "
        "--fleet-token to override, or set FLEET_URL and "
        "FLEET_API_TOKEN environment variables.",
    )
    fleet.add_argument(
        "--fleet-url",
        default="",
        metavar="URL",
        help=(
            "Fleet server URL (e.g., https://fleet.example.com). "
            "Resolution order: this flag, then $FLEET_URL, then the "
            "address from the fleetctl config file."
        ),
    )
    fleet.add_argument(
        "--fleet-token",
        default="",
        metavar="TOKEN",
        help=(
            "Fleet API token for authenticated API calls (team "
            "creation, host lookup, profile push, cleanup). "
            "Resolution order: this flag, then $FLEET_API_TOKEN, then "
            "the token from the fleetctl config file. To populate the "
            "config file, run: fleetctl login"
        ),
    )
    fleet.add_argument(
        "--fleetctl-context",
        default="default",
        metavar="CTX",
        help=(
            "Name of the fleetctl config context to read credentials "
            "from when --fleet-url or --fleet-token are not provided. "
            "(default: default)"
        ),
    )
    fleet.add_argument(
        "--fleetctl-path",
        default="./build/fleetctl",
        metavar="PATH",
        help=(
            "Path to the fleetctl binary. Used to build the agent "
            "package, apply team configurations, and run live queries. "
            "(default: ./build/fleetctl)"
        ),
    )
    fleet.add_argument(
        "--pkg-path",
        metavar="PATH",
        help=(
            "Path to a pre-built fleet-osquery.pkg. If provided, "
            "skips building a new package. WARNING: the pre-built "
            "package contains a baked-in enroll secret that may not "
            "match the test team this tool creates. The VM may enroll "
            "into a different team or fail to enroll. Only use this "
            "if you know the package was built with the correct secret."
        ),
    )

    # --- VM configuration ---
    vm = parser.add_argument_group(
        "VM configuration",
        "Control the tart VM used for testing.",
    )
    vm.add_argument(
        "--vm-name",
        default="",
        metavar="NAME",
        help=(
            "Name for the tart VM. If a VM with this name already "
            "exists, it will be deleted and recreated. "
            "(default: cis-test-macos-{version})"
        ),
    )
    vm.add_argument(
        "--tart-image",
        default="",
        metavar="IMAGE",
        help=(
            "Override the tart base image. By default, the image is "
            "chosen based on --macos-version: "
            "13=ghcr.io/cirruslabs/macos-ventura-base:latest, "
            "14=ghcr.io/cirruslabs/macos-sonoma-base:latest, "
            "15=ghcr.io/cirruslabs/macos-sequoia-base:latest, "
            "26=ghcr.io/cirruslabs/macos-tahoe-base:latest."
        ),
    )

    # --- Cleanup ---
    cleanup = parser.add_argument_group(
        "cleanup",
        "Control what happens after the test run completes.",
    )
    cleanup.add_argument(
        "--keep-vm",
        action="store_true",
        help=(
            "Do not delete the tart VM after the run, and reuse it "
            "on subsequent runs. If a VM with the same name already "
            "exists and is reachable, the tool skips creation and "
            "enrollment entirely, saving several minutes. Useful for "
            "iterating on tests. By default the VM is deleted when "
            "tests finish."
        ),
    )
    cleanup.add_argument(
        "--cleanup",
        action="store_true",
        help=(
            "Delete the Fleet team, the enrolled host record, and the "
            "tart VM after the run. Without this flag, the team and "
            "host remain in Fleet for inspection (the VM is still "
            "deleted unless --keep-vm is also set)."
        ),
    )

    # --- Timeouts and paths ---
    advanced = parser.add_argument_group("advanced")
    advanced.add_argument(
        "--query-timeout",
        type=int,
        default=60,
        metavar="SECS",
        help=(
            "Timeout in seconds for each fleetctl live query. "
            "Increase if queries are slow to return. (default: 60)"
        ),
    )
    advanced.add_argument(
        "--boot-timeout",
        type=int,
        default=180,
        metavar="SECS",
        help=(
            "Timeout in seconds for the VM to boot, get an IP, and "
            "complete fleet enrollment. Increase for slow networks "
            "or large VM images. (default: 180)"
        ),
    )
    advanced.add_argument(
        "--cis-dir",
        default="",
        metavar="PATH",
        help=(
            "Override the CIS directory path. By default, resolved "
            "from the repo root as ee/cis/macos-{version}/. Use this "
            "if running from outside the repo or testing a custom "
            "policy set."
        ),
    )
    advanced.add_argument(
        "--verbose",
        action="store_true",
        help=(
            "Show detailed output including SSH commands, query "
            "results, and script output."
        ),
    )

    return parser.parse_args()


def main() -> int:
    global VERBOSE
    args = parse_args()
    VERBOSE = args.verbose

    # Resolve Fleet URL and token: flag -> env var -> fleetctl config
    fleetctl_config = read_fleetctl_config(args.fleetctl_context)

    fleet_url = (
        args.fleet_url
        or os.environ.get("FLEET_URL", "")
        or fleetctl_config["address"]
    ).rstrip("/")

    fleet_token = (
        args.fleet_token
        or os.environ.get("FLEET_API_TOKEN", "")
        or fleetctl_config["token"]
    )

    if not fleet_url:
        log_error(
            "Fleet URL not found. Provide --fleet-url, set $FLEET_URL, "
            "or log in with: fleetctl login"
        )
        return 1
    if not fleet_token:
        log_error(
            "Fleet API token not found. Provide --fleet-token, set "
            "$FLEET_API_TOKEN, or log in with: fleetctl login"
        )
        return 1

    log(f"Using Fleet server: {fleet_url}")
    if fleet_url == fleetctl_config["address"] and not args.fleet_url:
        log_verbose(
            f"(credentials from fleetctl config, context: {args.fleetctl_context})"
        )

    fleetctl = args.fleetctl_path
    version_info = VERSION_MAP[args.macos_version]
    vm_name = args.vm_name or f"cis-test-macos-{args.macos_version}"
    tart_image = args.tart_image or version_info["image"]

    # Pre-flight: verify the Fleet API token is valid
    log("Verifying Fleet API token...")
    try:
        fleet_api(fleet_url, fleet_token, "GET", "/api/v1/fleet/me")
    except (HTTPError, URLError) as e:
        status = getattr(e, "code", None)
        if status == 401:
            log_error(
                "Fleet API token is invalid or expired. "
                "Please re-authenticate with: fleetctl login"
            )
        else:
            log_error(f"Cannot reach Fleet server at {fleet_url}: {e}")
        return 1

    # Resolve CIS directory
    repo_root = Path(__file__).resolve().parent.parent.parent
    cis_dir = Path(args.cis_dir) if args.cis_dir else repo_root / "ee" / "cis" / version_info["dir"]
    yaml_path = cis_dir / "cis-policy-queries.yml"
    scripts_dir = cis_dir / "test" / "scripts"
    profiles_dir = cis_dir / "test" / "profiles"

    if not yaml_path.exists():
        log_error(f"Policy file not found: {yaml_path}")
        return 1

    # Ensure tmp dir exists for temp files
    TMP_DIR.mkdir(parents=True, exist_ok=True)

    # Check prerequisites
    for tool in ["tart", "sshpass"]:
        if subprocess.run(["which", tool], capture_output=True).returncode != 0:
            log_error(f"Required tool not found: {tool}. Install it first.")
            return 1

    # Phase 1: Parse and filter policies
    log("Parsing CIS policies...")
    all_policies = parse_policies(yaml_path)
    log(f"Found {len(all_policies)} policies")

    cis_ids = args.cis_ids.split(",") if args.cis_ids else None
    match_terms = args.match.split(",") if args.match else None
    policies = filter_policies(all_policies, cis_ids, match_terms)
    log(f"Selected {len(policies)} policies for testing")

    if not policies:
        log_error("No policies matched the selection criteria")
        return 1

    ssh_breaking = SSH_BREAKING_CIS_IDS.get(args.macos_version, set())
    password_policy = PASSWORD_POLICY_CIS_IDS.get(args.macos_version, set())
    non_automatable = NON_AUTOMATABLE_CIS_IDS.get(args.macos_version, {})
    plans = build_test_plans(
        policies, scripts_dir, profiles_dir,
        ssh_breaking_ids=ssh_breaking,
        password_policy_ids=password_policy,
        non_automatable_ids=non_automatable,
    )

    # Determine which test types are allowed based on filter flags
    if args.only_scripts:
        allowed_types = {"PASS_FAIL", "PASS_ONLY"}
        skip_reason = "--only-scripts"
    elif args.only_mdm:
        allowed_types = {"PROFILE", "ORG_DECISION"}
        skip_reason = "--only-mdm"
    elif args.only_manual:
        allowed_types = {"MANUAL"}
        skip_reason = "--only-manual"
    else:
        allowed_types = None  # no type filter
        skip_reason = None

    # Apply filtering: type filter + --skip-manual
    skip_results = []
    active_plans = []
    for plan in plans:
        if allowed_types is not None and plan.test_type not in allowed_types:
            skip_results.append(
                TestResult(
                    plan.policy.cis_id,
                    plan.policy.name,
                    "SKIP",
                    f"{plan.test_type} excluded by {skip_reason}",
                )
            )
        elif args.only_scripts and (plan.policy.needs_mdm or plan.profiles):
            # --only-scripts should exclude anything requiring MDM,
            # even if it has scripts (the pass direction may depend
            # on a profile being present)
            skip_results.append(
                TestResult(
                    plan.policy.cis_id,
                    plan.policy.name,
                    "SKIP",
                    "requires MDM, excluded by --only-scripts",
                )
            )
        elif plan.test_type == "MANUAL" and args.skip_manual:
            skip_results.append(
                TestResult(
                    plan.policy.cis_id,
                    plan.policy.name,
                    "SKIP",
                    "manual test (--skip-manual)",
                )
            )
        else:
            active_plans.append(plan)

    any_needs_mdm = any(
        p.policy.needs_mdm
        or p.test_type == "PROFILE"
        or p.test_type == "ORG_DECISION"
        for p in active_plans
    )

    log(
        f"Test plans: {len(active_plans)} active, {len(skip_results)} skipped"
    )
    if any_needs_mdm:
        log("Some policies require MDM — will prompt for MDM enrollment")

    # Phase 2: Create Fleet team
    enroll_secret = str(uuid.uuid4())
    timestamp = time.strftime("%Y%m%d-%H%M%S")
    team_name = f"CIS-Test-macOS-{args.macos_version}-{timestamp}"
    team = create_fleet_team(fleet_url, fleet_token, fleetctl, team_name, enroll_secret)

    vm_proc = None
    results = list(skip_results)
    # Tracks fatal setup/runtime errors that happen outside of
    # per-test execution. print_summary alone returns 0 when `results`
    # has no FAIL/ERROR entries, which would mask a crash that killed
    # the run before any tests executed.
    fatal_exit_code = 0

    try:
        # Phase 3 & 4: VM setup and enrollment
        #
        # If --keep-vm is set and the VM already exists and is
        # reachable, reuse it instead of creating a new one. This
        # skips agent install and enrollment, saving several minutes.
        reused_vm = False
        needs_agent_install = True

        if args.keep_vm and (vm_is_running(vm_name) or vm_exists(vm_name)):
            if vm_is_running(vm_name):
                log(f"Attempting to reuse running VM: {vm_name}")
                ip = wait_for_ip(vm_name, timeout=30)
            else:
                log(f"Starting existing VM: {vm_name}")
                vm_proc = start_vm(vm_name)
                ip = wait_for_ip(vm_name, timeout=args.boot_timeout)

            # Check if SSH is reachable. If not (e.g., a previous test
            # disabled SSH), we must destroy and recreate the VM.
            try:
                wait_for_ssh(ip, timeout=30)
            except TimeoutError:
                log(
                    "SSH unreachable on existing VM (a previous test "
                    "may have disabled it). Recreating VM..."
                )
                delete_vm(vm_name)
                # Fall through to the fresh-VM path below
                args.keep_vm = False  # force fresh creation this run
            else:
                reused_vm = True
                if not vm_proc:
                    vm_proc = None  # not managed by us

                hostname = get_hostname(ip)
                team.hostname = hostname
                log(f"VM hostname: {hostname}")

                # Check if the agent is installed and the host is in Fleet
                result = ssh(ip, "cat /opt/orbit/identifier 2>/dev/null", timeout=10)
                has_identifier = result.returncode == 0 and result.stdout.strip()
                host_info = get_host_by_hostname(fleet_url, fleet_token, hostname) if has_identifier else None

                if host_info:
                    identifier = result.stdout.strip()
                    team.host_id = host_info.get("id")
                    log(f"Fleet host ID: {team.host_id}")
                    log("Host found in Fleet, skipping agent install")
                    needs_agent_install = False
                else:
                    log(
                        "Host not found in Fleet — will re-install agent "
                        "to enroll in the test team"
                    )

        if not reused_vm:
            # Fresh VM: create it first
            if args.pkg_path:
                pkg_path = Path(args.pkg_path)
                if not pkg_path.exists():
                    log_error(f"Package not found: {pkg_path}")
                    return 1
                log(
                    "WARNING: Using pre-built package. The VM will enroll "
                    "with whatever secret was baked into this package, which "
                    "may not match the test team. If the host does not appear "
                    "in the test team, rebuild without --pkg-path."
                )
            else:
                pkg_path = build_fleet_pkg(fleet_url, team.enroll_secret, fleetctl)

            create_vm(vm_name, tart_image)
            vm_proc = start_vm(vm_name)

            log("Waiting for VM to boot...")
            ip = wait_for_ip(vm_name, timeout=args.boot_timeout)
            wait_for_ssh(ip, timeout=60)

            hostname = get_hostname(ip)
            team.hostname = hostname
            log(f"VM hostname: {hostname}")

        # Install agent if needed (fresh VM or reused VM without valid enrollment)
        if needs_agent_install:
            if not reused_vm:
                # pkg already built above for fresh VMs
                pass
            else:
                # Reused VM needs a new agent to enroll in the test team
                if args.pkg_path:
                    pkg_path = Path(args.pkg_path)
                else:
                    pkg_path = build_fleet_pkg(fleet_url, team.enroll_secret, fleetctl)

            install_agent(ip, pkg_path)
            identifier = wait_for_identifier(ip, timeout=args.boot_timeout)
            wait_for_fleet_registration(fleet_url, identifier, timeout=args.boot_timeout)

            # Look up host ID in Fleet
            time.sleep(10)  # Give Fleet a moment to index the host
            host_info = get_host_by_hostname(fleet_url, fleet_token, hostname)
            if host_info:
                team.host_id = host_info.get("id")
                log(f"Fleet host ID: {team.host_id}")

        # Validate the host landed in the correct team; if not,
        # transfer it. This happens when the VM was previously
        # enrolled with a different secret (e.g., --keep-vm with a
        # host that's already MDM-enrolled from a prior run).
        if team.host_id and team.team_id:
            host_info = get_host_by_hostname(fleet_url, fleet_token, hostname)
            if host_info:
                host_team_id = host_info.get("team_id") or host_info.get("fleet_id")
                if host_team_id == team.team_id:
                    log(f"Host is in the correct team (ID: {team.team_id})")
                else:
                    current = f"team {host_team_id}" if host_team_id else "no team"
                    log(
                        f"Host is in {current}, expected {team.team_id} "
                        f"({team.name}). Transferring..."
                    )
                    try:
                        transfer_host_to_team(
                            fleet_url, fleet_token, team.host_id, team.team_id
                        )
                        log(f"Host transferred to team {team.team_id}")
                    except HTTPError:
                        log_error(
                            "Failed to transfer host. Profile-based "
                            "tests will fail."
                        )

        # Phase 5: MDM enrollment and profile delivery
        pre_profile_passed = set()
        if any_needs_mdm:
            # Check if the host is already MDM-enrolled; if so, skip
            # the interactive enrollment step. Poll a few times since
            # Fleet may take a moment to refresh MDM status after
            # an agent re-install.
            already_mdm = False
            for attempt in range(6):
                host_info = get_host_by_hostname(fleet_url, fleet_token, hostname)
                mdm = (host_info or {}).get("mdm", {}) or {}
                mdm_status = mdm.get("enrollment_status")
                if (
                    mdm_status
                    and "On" in str(mdm_status)
                    and mdm.get("connected_to_fleet")
                ):
                    already_mdm = True
                    log(
                        f"Host already MDM-enrolled ({mdm_status}), "
                        "skipping MDM enrollment prompt"
                    )
                    break
                if attempt < 5:
                    log_verbose(
                        f"MDM status check {attempt + 1}/6: "
                        f"{mdm_status!r}, retrying..."
                    )
                    time.sleep(5)

            if not already_mdm:
                enroll_mdm(ip, fleet_url, identifier)

            # Before pushing profiles, verify that profile-only
            # policies currently fail (no profiles installed yet).
            # This confirms the query can actually detect non-compliance.
            #
            # First, clear the team's profile list and wait for the
            # host to process the removal. This avoids stale data from
            # prior test runs causing the pre-profile check to report
            # "already passes" for policies whose values still linger
            # in managed_policies.
            profile_plans = [p for p in active_plans if p.test_type == "PROFILE"]
            pre_profile_passed = set()
            if profile_plans:
                log("Clearing team profiles to get a clean baseline...")
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name, []
                )
                time.sleep(20)  # Wait for the host to process removal

                log(f"Verifying {len(profile_plans)} profile-only queries fail before delivery...")
                for plan in profile_plans:
                    passes = run_query(plan.policy.query, hostname, fleetctl, args.query_timeout)
                    if passes:
                        pre_profile_passed.add(plan.policy.cis_id)
                        log(f"  [!] CIS {plan.policy.cis_id}: already passes (may not detect non-compliance)")
                    else:
                        log(f"  [ok] CIS {plan.policy.cis_id}: fails as expected")
                log(
                    f"Pre-profile check complete: "
                    f"{len(profile_plans) - len(pre_profile_passed)} fail as expected, "
                    f"{len(pre_profile_passed)} already pass"
                )

            # Collect profiles needed by PROFILE plans (tested by
            # presence during the test). Exclusions:
            #   - ORG_DECISION: toggles its own enable/disable profiles
            #   - PASS_FAIL with MDM: toggles its own profile
            #   - MANUAL: includes quarantined plans
            #     (SSH_BREAKING_CIS_IDS, PASSWORD_POLICY_CIS_IDS,
            #     NON_AUTOMATABLE_CIS_IDS) whose mobileconfigs must
            #     NOT be bulk-pushed. Password-policy profiles break
            #     VM SSH auth; the Siri/iCloud profiles cause cross-
            #     test state pollution; 2.6.3 uses wrong keys. These
            #     profiles are discovered by _discover_profiles() but
            #     deliberately kept out of the bulk push.
            all_profiles = []
            for plan in active_plans:
                if plan.test_type == "ORG_DECISION":
                    continue
                if plan.test_type == "PASS_FAIL" and plan.policy.needs_mdm and plan.profiles:
                    continue
                if plan.test_type == "MANUAL":
                    continue
                all_profiles.extend(plan.profiles)
            base_profiles = list({str(p): p for p in all_profiles}.values())
            if base_profiles:
                push_profiles_to_team(
                    fleet_url, fleet_token, fleetctl, team_name, base_profiles
                )
                log("Waiting for profiles to be delivered...")
                time.sleep(30)
        else:
            base_profiles = []

        # Phase 6: Test execution
        log(f"Running {len(active_plans)} test(s)...")
        for plan in active_plans:
            result = run_test(
                plan, ip, hostname, fleetctl, args.query_timeout,
                pre_profile_passed=pre_profile_passed,
                fleet_url=fleet_url,
                fleet_token=fleet_token,
                team_name=team_name,
                base_profiles=base_profiles,
            )
            results.append(result)
            status_symbol = {
                "PASS": "+", "FAIL": "x", "SKIP": "-", "ERROR": "!"
            }.get(result.status, "?")
            log(f"  [{status_symbol}] CIS {result.cis_id}: {result.status}")


    except KeyboardInterrupt:
        log("\nInterrupted by user")
        fatal_exit_code = 130
    except Exception as e:
        log_error(f"Fatal error: {e}")
        if VERBOSE:
            import traceback
            traceback.print_exc()
        fatal_exit_code = 1
    finally:
        # Phase 8: Cleanup
        if args.cleanup:
            if team.host_id:
                delete_fleet_host(fleet_url, fleet_token, team.host_id)
            if team.team_id:
                delete_fleet_team(fleet_url, fleet_token, team.team_id)
            # Delete the VM even if this process didn't start it
            # (e.g., --keep-vm path that reused a running VM). The
            # --cleanup flag promises to clean up the VM; ownership
            # tracking via vm_proc was incorrect.
            delete_vm(vm_name)
        elif not args.keep_vm:
            if vm_proc:
                log("Cleaning up VM...")
                delete_vm(vm_name)

    # Phase 7: Report
    return max(fatal_exit_code, print_summary(results, team))


if __name__ == "__main__":
    sys.exit(main())
