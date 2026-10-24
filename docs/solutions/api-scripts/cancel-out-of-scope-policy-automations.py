#!/usr/bin/env python3
"""Cancel queued policy automations on hosts that are no longer in the policy's scope.

When a policy's labels (or platform) change, hosts that drop out of scope can keep
running the policy's script or software automation: anything already queued still
runs, and each failed run queues a retry (up to 3 attempts). This script finds
upcoming activities tied to a policy that no longer applies to the host and cancels
them.

A policy is "in scope" for a host when Fleet lists it on the host's details
(GET /api/v1/fleet/hosts/:id), so the label/platform logic is Fleet's own.

Canceling a script that's already running doesn't stop it on the host, but Fleet
discards its result, so no retry is queued.

Requires Fleet 4.87+ and an API token for a user that can cancel host activities
(admin or maintainer on the fleet). Uses only the Python standard library.

Usage:
  export FLEET_URL=https://fleet.example.com
  export FLEET_API_TOKEN=...

  # Dry run (default): list what would be canceled
  ./cancel-out-of-scope-policy-automations.py --fleet-id 5

  # Cancel, limited to specific policies
  ./cancel-out-of-scope-policy-automations.py --fleet-id 5 --policy-id 123 --policy-id 456 --apply

  # Cancel, then keep sweeping for 15 minutes to catch retries queued by runs
  # that were already in progress
  ./cancel-out-of-scope-policy-automations.py --fleet-id 5 --apply --watch 15
"""

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor

# Upcoming activity types that can come from a policy automation. App Store (VPP)
# installs aren't included: Fleet doesn't report a policy_id on those upcoming
# activities, so they can't be matched to a policy.
POLICY_AUTOMATION_TYPES = {"ran_script", "installed_software"}


class APIError(RuntimeError):
    def __init__(self, message, status=None):
        super().__init__(message)
        self.status = status


class Fleet:
    def __init__(self, url, token):
        self.url = url.rstrip("/")
        self.token = token

    def request(self, method, path, params=None):
        url = self.url + path
        if params:
            url += "?" + urllib.parse.urlencode(params)
        req = urllib.request.Request(url, method=method)
        req.add_header("Authorization", f"Bearer {self.token}")
        last = None
        for attempt in range(5):
            try:
                with urllib.request.urlopen(req, timeout=60) as resp:
                    body = resp.read()
                    return json.loads(body) if body else {}
            except urllib.error.HTTPError as e:
                detail = e.read().decode(errors="replace")[:300]
                last = APIError(f"{method} {path} -> HTTP {e.code}: {detail}", e.code)
                if e.code == 429 or e.code >= 500:
                    time.sleep(2 ** attempt)
                    continue
                raise last from None
            except urllib.error.URLError as e:
                last = APIError(f"{method} {path} -> {e.reason}")
                time.sleep(2 ** attempt)
        raise APIError(f"{last} (gave up after 5 attempts)", last.status)

    def hosts(self, fleet_id):
        page = 0
        while True:
            params = {"per_page": 500, "page": page}
            if fleet_id is not None:
                params["team_id"] = fleet_id
            batch = self.request("GET", "/api/v1/fleet/hosts", params).get("hosts", [])
            yield from batch
            if len(batch) < 500:
                return
            page += 1

    def upcoming(self, host_id):
        page, out = 0, []
        while True:
            data = self.request(
                "GET", f"/api/v1/fleet/hosts/{host_id}/activities/upcoming",
                {"per_page": 100, "page": page},
            )
            out.extend(data.get("activities") or [])
            if not (data.get("meta") or {}).get("has_next_results"):
                return out
            page += 1

    def in_scope_policy_ids(self, host_id):
        host = self.request(
            "GET", f"/api/v1/fleet/hosts/{host_id}", {"exclude_software": "true"}
        )["host"]
        if host.get("policies") is None:
            # Without the host's policy list we can't tell what's in scope, so don't
            # treat it as "no policies" (that would cancel every policy automation).
            raise APIError(f"host {host_id}: details have no policies list, skipping")
        return {p["id"] for p in host["policies"]}

    def cancel(self, host_id, activity_id):
        self.request(
            "DELETE", f"/api/v1/fleet/hosts/{host_id}/activities/upcoming/{activity_id}"
        )


def check_host(fleet, host, policy_filter, apply):
    """Return a list of (host, activity, result) for out-of-scope policy automations."""
    pending = [
        a for a in fleet.upcoming(host["id"])
        if a.get("type") in POLICY_AUTOMATION_TYPES
        and (a.get("details") or {}).get("policy_id") is not None
        and (not policy_filter or a["details"]["policy_id"] in policy_filter)
    ]
    if not pending:
        return []

    in_scope = fleet.in_scope_policy_ids(host["id"])
    results = []
    for a in pending:
        if a["details"]["policy_id"] in in_scope:
            continue
        result = "would cancel"
        if apply:
            try:
                fleet.cancel(host["id"], a["uuid"])
                result = "canceled"
            except APIError as e:
                # 404: the activity started or finished between listing and canceling.
                result = "already gone" if e.status == 404 else f"error: {e}"
        results.append((host, a, result))
    return results


def sweep(fleet, args, policy_filter):
    """Run one sweep. Returns the number of hosts or cancels that failed."""
    hosts = list(fleet.hosts(args.fleet_id))
    found, failed_hosts = [], 0
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = [pool.submit(check_host, fleet, h, policy_filter, args.apply) for h in hosts]
        for f in futures:
            try:
                found.extend(f.result())
            except Exception as e:  # one bad host shouldn't stop the sweep
                failed_hosts += 1
                print(f"warning: {e}", file=sys.stderr)

    counts = {}
    for host, a, result in found:
        d = a["details"]
        name = d.get("script_name") or d.get("software_title") or a.get("type")
        print(
            f"{result:<13} host={host['id']} ({host.get('display_name') or host.get('hostname')}) "
            f"policy={d['policy_id']} ({d.get('policy_name')}) {a['type']}={name} "
            f"activity={a['uuid']}"
        )
        key = "error" if result.startswith("error") else result
        counts[key] = counts.get(key, 0) + 1
    summary = ", ".join(f"{n} {k}" for k, n in sorted(counts.items())) or "nothing to cancel"
    if not args.apply and found:
        summary += " (dry run, use --apply to cancel)"
    if failed_hosts:
        summary += f", {failed_hosts} hosts skipped after errors"
    print(f"checked {len(hosts)} hosts: {summary}")
    return failed_hosts + counts.get("error", 0)


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--fleet-id", type=int, help="only check hosts in this fleet (default: all hosts)")
    p.add_argument("--policy-id", type=int, action="append", default=[],
                   help="only cancel automations for this policy (repeatable)")
    p.add_argument("--apply", action="store_true", help="cancel (default is a dry run)")
    p.add_argument("--watch", type=float, default=0, metavar="MINUTES",
                   help="keep sweeping for this many minutes to catch retries")
    p.add_argument("--interval", type=int, default=60, metavar="SECONDS",
                   help="seconds between sweeps with --watch (default 60)")
    p.add_argument("--workers", type=int, default=8, help="concurrent API requests (default 8)")
    args = p.parse_args()

    url, token = os.environ.get("FLEET_URL"), os.environ.get("FLEET_API_TOKEN")
    if not url or not token:
        sys.exit("set FLEET_URL and FLEET_API_TOKEN")

    fleet = Fleet(url, token)
    policy_filter = set(args.policy_id)
    deadline = time.time() + args.watch * 60
    failures = 0
    while True:
        print(f"== sweep at {time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())}")
        try:
            failures += sweep(fleet, args, policy_filter)
        except APIError as e:
            sys.exit(f"error: {e}")
        if time.time() + args.interval > deadline:
            break
        time.sleep(args.interval)
    if failures:
        sys.exit(f"finished with {failures} errors, see output above")


if __name__ == "__main__":
    main()
