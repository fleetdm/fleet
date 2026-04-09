#!/usr/bin/env python3
"""Sync device-to-user assignments from Jamf Pro to Fleet.

For each computer in Jamf this script:
  1. Reads the assigned user's email (falls back to username if no email is set).
  2. Finds the matching host in Fleet by serial number.
  3. Sets the IDP username on that Fleet host via the API.

Required environment variables
-------------------------------
  JAMF_URL             Base URL of your Jamf Pro server
                       (e.g. https://org.jamfcloud.com)

  --- Jamf auth: choose ONE of the following two options ---

  Option A – OAuth 2.0 client credentials (Jamf Pro 10.49+, recommended):
  JAMF_CLIENT_ID       Jamf Pro API client ID
  JAMF_CLIENT_SECRET   Jamf Pro API client secret

  Option B – username / password:
  JAMF_USERNAME        Jamf Pro username
  JAMF_PASSWORD        Jamf Pro password

  --- Fleet ---
  FLEET_URL            Base URL of your Fleet server
                       (e.g. https://fleet.example.com)
  FLEET_API_TOKEN      Fleet API token (Settings → My account → Get API token)

Usage
-----
  # 1. Install dependencies (one-time)
  pip install requests

  # 2. Export credentials
  export JAMF_URL="https://org.jamfcloud.com"

  # Option A – OAuth (Jamf Pro 10.49+, recommended)
  export JAMF_CLIENT_ID="your-client-id"
  export JAMF_CLIENT_SECRET="your-client-secret"

  # Option B – username / password
  export JAMF_USERNAME="your-jamf-username"
  export JAMF_PASSWORD="your-jamf-password"

  export FLEET_URL="https://fleet.example.com"
  export FLEET_API_TOKEN="your-fleet-api-token"

  # 3. Dry run first to preview changes without modifying Fleet
  python3 sync_jamf_users_to_fleet.py --dry-run

  # 4. Run for real
  python3 sync_jamf_users_to_fleet.py
"""

import argparse
import os
import sys
import time

import requests


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

JAMF_URL = os.getenv("JAMF_URL", "").rstrip("/")
JAMF_CLIENT_ID = os.getenv("JAMF_CLIENT_ID", "")
JAMF_CLIENT_SECRET = os.getenv("JAMF_CLIENT_SECRET", "")
JAMF_USERNAME = os.getenv("JAMF_USERNAME", "")
JAMF_PASSWORD = os.getenv("JAMF_PASSWORD", "")

FLEET_URL = os.getenv("FLEET_URL", "").rstrip("/")
FLEET_API_TOKEN = os.getenv("FLEET_API_TOKEN", "")

# Small delay between Fleet write calls to avoid hitting rate limits
FLEET_REQUEST_DELAY = 0.05  # seconds


# ---------------------------------------------------------------------------
# Startup validation
# ---------------------------------------------------------------------------

def _die(msg):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(1)


if not JAMF_URL:
    _die("JAMF_URL is not set.")

if not FLEET_URL:
    _die("FLEET_URL is not set.")

if not FLEET_API_TOKEN:
    _die("FLEET_API_TOKEN is not set.")

_have_oauth = bool(JAMF_CLIENT_ID and JAMF_CLIENT_SECRET)
_have_basic = bool(JAMF_USERNAME and JAMF_PASSWORD)

if not _have_oauth and not _have_basic:
    _die(
        "Jamf credentials are not set. "
        "Provide JAMF_CLIENT_ID + JAMF_CLIENT_SECRET (preferred) "
        "or JAMF_USERNAME + JAMF_PASSWORD."
    )


# ---------------------------------------------------------------------------
# Jamf authentication
# ---------------------------------------------------------------------------

_jamf_token: str = ""
_jamf_token_expires_at: float = 0.0


def _refresh_jamf_token() -> str:
    """Obtain or renew a Jamf Pro bearer token and return it."""
    global _jamf_token, _jamf_token_expires_at

    if _have_oauth:
        # OAuth 2.0 client credentials (Jamf Pro 10.49+)
        resp = requests.post(
            f"{JAMF_URL}/api/oauth/token",
            data={
                "grant_type": "client_credentials",
                "client_id": JAMF_CLIENT_ID,
                "client_secret": JAMF_CLIENT_SECRET,
            },
            headers={"Accept": "application/json"},
            timeout=30,
        )
        resp.raise_for_status()
        body = resp.json()
        _jamf_token = body["access_token"]
        _jamf_token_expires_at = time.time() + body.get("expires_in", 1800)
    else:
        # Basic-auth token exchange (Jamf Pro 10.35+)
        resp = requests.post(
            f"{JAMF_URL}/api/v1/auth/token",
            auth=(JAMF_USERNAME, JAMF_PASSWORD),
            headers={"Accept": "application/json"},
            timeout=30,
        )
        resp.raise_for_status()
        body = resp.json()
        _jamf_token = body["token"]
        _jamf_token_expires_at = time.time() + 1800  # default 30-min lifetime

    return _jamf_token


def _jamf_headers() -> dict:
    """Return request headers with a valid Jamf bearer token."""
    global _jamf_token, _jamf_token_expires_at
    # Refresh 60 s before expiry to avoid mid-run failures
    if not _jamf_token or time.time() >= _jamf_token_expires_at - 60:
        _refresh_jamf_token()
    return {
        "Authorization": f"Bearer {_jamf_token}",
        "Accept": "application/json",
    }


# ---------------------------------------------------------------------------
# Jamf helpers
# ---------------------------------------------------------------------------

def get_all_jamf_computers():
    """Yield computer records from Jamf (id + serial_number + user).

    Uses the Classic API /subset/basic endpoint, which returns serial_number,
    email_address, and username for every computer in a single request —
    avoiding a separate per-device location fetch.
    """
    resp = requests.get(
        f"{JAMF_URL}/JSSResource/computers/subset/basic",
        headers=_jamf_headers(),
        timeout=60,
    )
    resp.raise_for_status()
    computers = resp.json().get("computers", [])

    for c in computers:
        email = (c.get("email_address") or "").strip()
        username = (c.get("username") or "").strip()
        yield {
            "jamf_id": c["id"],
            "serial_number": (c.get("serial_number") or "").strip(),
            "user": email or username or None,
        }


# ---------------------------------------------------------------------------
# Fleet helpers
# ---------------------------------------------------------------------------

_fleet_headers = {
    "Authorization": f"Bearer {FLEET_API_TOKEN}",
    "Content-Type": "application/json",
}


def get_fleet_host_by_serial(serial: str) -> dict | None:
    """Look up a Fleet host by serial number. Returns the host dict or None."""
    resp = requests.get(
        f"{FLEET_URL}/api/v1/fleet/hosts/identifier/{serial}",
        headers=_fleet_headers,
        timeout=30,
    )
    if resp.status_code == 404:
        return None
    resp.raise_for_status()
    return resp.json().get("host")


def assign_fleet_device_mapping(host_id: int, email: str) -> None:
    """Set the IDP username for a Fleet host."""
    resp = requests.put(
        f"{FLEET_URL}/api/v1/fleet/hosts/{host_id}/device_mapping",
        headers=_fleet_headers,
        json={"email": email, "source": "idp"},
        timeout=30,
    )
    resp.raise_for_status()


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Log what would be assigned without making any changes in Fleet.",
    )
    args = parser.parse_args()
    dry_run = args.dry_run

    if dry_run:
        print("DRY RUN — no changes will be made in Fleet.")
        print()

    assigned = 0
    skipped_no_serial = 0
    skipped_no_user = 0
    skipped_not_in_fleet = 0
    errors = 0

    print(f"Fetching computers from Jamf Pro ({JAMF_URL})…")

    for computer in get_all_jamf_computers():
        serial = computer["serial_number"]
        jamf_id = computer["jamf_id"]

        if not serial:
            skipped_no_serial += 1
            continue

        user = computer["user"]

        if not user:
            print(f"  [SKIP] {serial}: no user assigned in Jamf")
            skipped_no_user += 1
            continue

        # --- Look up the host in Fleet by serial number ---
        try:
            host = get_fleet_host_by_serial(serial)
        except requests.HTTPError as exc:
            print(f"  [WARN] Fleet error looking up serial {serial}: {exc}")
            errors += 1
            continue

        if host is None:
            print(f"  [SKIP] {serial}: not found in Fleet")
            skipped_not_in_fleet += 1
            continue

        # --- Assign the user in Fleet ---
        if dry_run:
            print(f"  [DRY]  {serial} → {user} (Fleet host ID {host['id']})")
            assigned += 1
        else:
            try:
                assign_fleet_device_mapping(host["id"], user)
                print(f"  [OK]   {serial} → {user} (Fleet host ID {host['id']})")
                assigned += 1
            except requests.HTTPError as exc:
                print(f"  [WARN] Fleet error assigning {user} to host {host['id']}: {exc}")
                errors += 1

            time.sleep(FLEET_REQUEST_DELAY)

    print()
    print("Done (dry run — no changes were made)." if dry_run else "Done.")
    print(f"  {'Would assign' if dry_run else 'Assigned'}                   : {assigned}")
    print(f"  Skipped (no serial in Jamf): {skipped_no_serial}")
    print(f"  Skipped (no user in Jamf)  : {skipped_no_user}")
    print(f"  Skipped (not in Fleet)     : {skipped_not_in_fleet}")
    print(f"  Errors                     : {errors}")


if __name__ == "__main__":
    main()
