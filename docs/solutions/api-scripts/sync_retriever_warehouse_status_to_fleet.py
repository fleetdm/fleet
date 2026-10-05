#!/usr/bin/env python3
"""Sync each device's Retriever warehouse status into a Fleet custom host vital.

For each device in Retriever's warehouse, this script:
  1. Finds the Fleet host or hosts with the device's serial number.
  2. Reads the host's current value for the custom host vital.
  3. Writes Retriever's status only when it changed, using Retriever's display
     name (e.g. "Ready For Deployment") rather than the raw status code.

It never clears a value, and it only sends GET requests to Retriever.

Setup guide: https://fleetdm.com/guides/sync-warehouse-status-from-retriever

Environment variables
---------------------
  FLEET_URL          Base URL of your Fleet server (e.g. https://fleet.example.com)
  FLEET_API_TOKEN    Token for an API-only user with the global Maintainer role,
                     limited to these endpoints:
                       GET /api/v1/fleet/hosts
                       GET /api/v1/fleet/hosts/:id
                       PUT /api/v1/fleet/hosts/:host_id/custom_host_vitals/:id
  RETRIEVER_API_KEY  Retriever API key. Retriever keys can also submit billable
                     orders, so store it like a password.
  VITAL_NAME         Optional. The custom host vital's name (default: "Warehouse status").
  DRY_RUN            Optional. Set to 1 to print changes without writing them.

Usage
-----
  DRY_RUN=1 python3 sync_retriever_warehouse_status_to_fleet.py
  python3 sync_retriever_warehouse_status_to_fleet.py

Requires Python 3. No third-party packages.
"""
import json, os, sys, urllib.error, urllib.parse, urllib.request

FLEET_URL = os.environ["FLEET_URL"].rstrip("/")  # e.g. https://fleet.example.com
FLEET_TOKEN = os.environ["FLEET_API_TOKEN"]
RETRIEVER_KEY = os.environ["RETRIEVER_API_KEY"]
VITAL_NAME = os.environ.get("VITAL_NAME", "Warehouse status")
DRY_RUN = os.environ.get("DRY_RUN") == "1"
RETRIEVER_URL = "https://app.helloretriever.com/api/v2/warehouse/"

# Retriever's display labels, where they differ from the title-cased status code.
LABELS = {
    "delivered_address": "Delivered For Disposal",
    "deployment_initiated": "Deployment In Transit",
    "retrieval_initiated": "Retrieval In Transit",
    "transfer_ownership": "Transferred Ownership",
}


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None  # urllib would send the Authorization header to the redirect target, even on another host.


OPENER = urllib.request.build_opener(NoRedirect)


def call(method, url, token, body=None):
    req = urllib.request.Request(
        url,
        method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "User-Agent": "retriever-warehouse-sync",
        },
    )
    with OPENER.open(req, timeout=60) as resp:
        raw = resp.read()
        return json.loads(raw) if raw else {}


# 1. Read every device from Retriever. GET only, 50 devices per page.
statuses, url = {}, RETRIEVER_URL
while url:
    if not url.startswith(RETRIEVER_URL):
        raise SystemExit(f"Unexpected pagination URL, not sending the key there: {url}")
    page = call("GET", url, RETRIEVER_KEY)
    for device in page["results"]:
        serial = (device.get("serial_number") or "").strip().upper()
        code = device.get("status")
        if serial and code and serial not in statuses:  # Newest record wins.
            statuses[serial] = LABELS.get(code, code.replace("_", " ").title())
    url = page.get("next")

# 2. Update each matching Fleet host, only when its value changed.
failed = False
for serial, label in sorted(statuses.items()):
    try:
        query = urllib.parse.urlencode({"query": serial})
        for match in call("GET", f"{FLEET_URL}/api/v1/fleet/hosts?{query}", FLEET_TOKEN)["hosts"]:
            if (match.get("hardware_serial") or "").strip().upper() != serial:
                continue  # The search also matches hostnames and other fields.
            host = call("GET", f"{FLEET_URL}/api/v1/fleet/hosts/{match['id']}", FLEET_TOKEN)["host"]
            vital = next((v for v in host.get("custom_host_vitals") or [] if v["name"] == VITAL_NAME), None)
            if vital is None:
                raise SystemExit(f"Custom host vital {VITAL_NAME!r} doesn't exist in Fleet")
            if vital["value"] == label:
                continue
            print(f"{host['display_name']} ({serial}): {vital['value'] or '(empty)'} -> {label}")
            if not DRY_RUN:
                path = f"/api/v1/fleet/hosts/{host['id']}/custom_host_vitals/{vital['custom_host_vital_id']}"
                call("PUT", FLEET_URL + path, FLEET_TOKEN, {"value": label})
    except urllib.error.HTTPError as e:
        print(f"{serial}: {e} from {e.filename}: {e.read().decode(errors='replace')[:500]}", file=sys.stderr)
        failed = True

sys.exit(1 if failed else 0)
