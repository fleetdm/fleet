#!/usr/bin/env python3
"""Sync each Apple device's AppleCare and warranty end date into a Fleet custom host vital.

For each device in Apple Business (or Apple Business Manager), this script:
  1. Finds the Fleet host or hosts with the device's serial number.
  2. Reads the device's coverage from the Apple Business API and takes the latest
     end date across its non-canceled coverage, including coverage that already
     ended. Devices with no dated coverage, like AppleCare+ for Business
     subscriptions, are skipped.
  3. Writes the date as YYYY-MM-DD, only when the host's value changed.

It never clears a value, and it only sends GET requests to the Apple Business API.

Setup guide: https://fleetdm.com/guides/sync-applecare-coverage-from-apple-business

Environment variables
---------------------
  FLEET_URL             HTTPS base URL of your Fleet server (e.g. https://fleet.example.com)
  FLEET_API_TOKEN       Token for an API-only user with the global Maintainer role,
                        limited to these endpoints:
                          GET /api/v1/fleet/hosts
                          GET /api/v1/fleet/hosts/:id
                          PUT /api/v1/fleet/hosts/:host_id/custom_host_vitals/:id
  ABM_CLIENT_ASSERTION  Client assertion for an Apple Business API account, signed
                        with sign_apple_business_client_assertion.py. It's valid for
                        up to 180 days. The script warns 30 days before it expires.
  VITAL_NAME            Optional. The custom host vital's name
                        (default: "AppleCare warranty expiration").
  DRY_RUN               Optional. Set to 1 to print changes without writing them.

Usage
-----
  DRY_RUN=1 python3 sync_applecare_coverage_to_fleet.py
  python3 sync_applecare_coverage_to_fleet.py

Exits 0 on success, 1 if some devices failed, and 2 on a fatal error like an
expired assertion or a 401 or 403 from either API.

Requires Python 3. No third-party packages.
"""
import base64, json, os, sys, time, urllib.error, urllib.parse, urllib.request

APPLE_TOKEN_URL = "https://account.apple.com/auth/oauth2/token"
APPLE_API_URL = "https://api-business.apple.com/v1"
ASSERTION_WARN_DAYS = 30
TOKEN_MAX_AGE = 50 * 60  # Apple access tokens last one hour.
MAX_PAGES = 1000


class SyncError(Exception):
    def __init__(self, message, code=None):
        super().__init__(message)
        self.code = code


class FatalError(Exception):
    pass


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args):
        return None  # urllib would send the Authorization header to the redirect target, even on another host.


OPENER = urllib.request.build_opener(NoRedirect)


def call(method, url, token=None, body=None):
    """Sends a request, retrying 429 and 5xx responses up to twice."""
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Accept": "application/json", "User-Agent": "fleet-applecare-sync"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if data is not None:
        headers["Content-Type"] = "application/json"
    elif method == "POST":
        headers["Content-Type"] = "application/x-www-form-urlencoded"
    # Leave the query string out of errors: the token request carries the client assertion there.
    where = f"{method} {url.split('?')[0]}"
    last, code, wait = None, None, 0
    for attempt in range(3):
        time.sleep(wait)
        req = urllib.request.Request(url, data=data, method=method, headers=headers)
        try:
            with OPENER.open(req, timeout=60) as resp:
                raw = resp.read()
                return json.loads(raw) if raw else {}
        except urllib.error.HTTPError as e:
            code = e.code
            last = f"HTTP {e.code} from {where}: {e.read()[:300].decode('utf-8', 'replace')}"
            if e.code != 429 and e.code < 500:
                break
            retry_after = (e.headers.get("Retry-After") or "").strip() if e.code == 429 else ""
            wait = min(int(retry_after), 120) if retry_after.isdigit() else 10 * (attempt + 1)
        except (urllib.error.URLError, TimeoutError) as e:
            code, last, wait = None, f"network error on {where}: {e}", 10 * (attempt + 1)
    raise SyncError(last, code)


def assertion_claims(assertion):
    """Returns the client ID (the assertion's sub) and its expiry time."""
    try:
        payload = assertion.split(".")[1]
        claims = json.loads(base64.urlsafe_b64decode(payload + "=" * (-len(payload) % 4)))
        return str(claims["sub"]), int(claims["exp"])
    except (IndexError, KeyError, TypeError, ValueError):
        raise FatalError("ABM_CLIENT_ASSERTION isn't a client assertion JWT. Sign one with sign_apple_business_client_assertion.py.")


class Apple:
    def __init__(self, client_id, assertion):
        self.client_id, self.assertion = client_id, assertion
        self.token, self.fetched = None, 0

    def refresh(self):
        # Apple documents these parameters in the query string, not the body.
        query = urllib.parse.urlencode({
            "grant_type": "client_credentials",
            "client_id": self.client_id,
            "client_assertion_type": "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
            "client_assertion": self.assertion,
            "scope": "business.api",
        })
        try:
            self.token = call("POST", f"{APPLE_TOKEN_URL}?{query}")["access_token"]
        except (SyncError, KeyError) as e:
            raise FatalError(f"Apple Business token request failed. If the API account's key was revoked, sign a new assertion. {e}")
        self.fetched = time.time()

    def get(self, url):
        if not self.token or time.time() - self.fetched > TOKEN_MAX_AGE:
            self.refresh()
        try:
            return call("GET", url, self.token)
        except SyncError as e:
            # Apple can reject a token before it expires and asks clients to get a new one on any 401.
            if e.code != 401:
                raise
            self.refresh()
            return call("GET", url, self.token)

    def pages(self, url, what):
        """Yields each page of a list, following links.next."""
        for _ in range(MAX_PAGES):
            page = self.get(url)
            yield page
            url = (page.get("links") or {}).get("next")
            if not url:
                return
            if not url.startswith(APPLE_API_URL + "/"):
                raise FatalError(f"Unexpected pagination URL, not sending the token there: {url}")
        raise FatalError(f"Apple Business returned more than {MAX_PAGES} pages of {what}")


def apple_devices(apple):
    """Returns {SERIAL: device ID} for devices that are still in the organization."""
    devices = {}
    for page in apple.pages(f"{APPLE_API_URL}/orgDevices", "devices"):
        for d in page.get("data") or []:
            attrs = d.get("attributes") or {}
            serial = (attrs.get("serialNumber") or d.get("id") or "").strip().upper()
            if serial and d.get("id") and not attrs.get("releasedFromOrgDateTime"):
                devices[serial] = d["id"]
    return devices


def fleet_hosts_by_serial(fleet_url, token):
    """Returns {SERIAL: [host IDs]}. Several hosts can share a serial, for example after a re-enrollment."""
    hosts_by_serial = {}
    for page in range(MAX_PAGES):
        query = urllib.parse.urlencode({"page": page, "per_page": 500, "order_key": "id"})
        hosts = call("GET", f"{fleet_url}/api/v1/fleet/hosts?{query}", token).get("hosts") or []
        if not hosts:
            return hosts_by_serial
        for h in hosts:
            serial = (h.get("hardware_serial") or "").strip().upper()
            if serial:
                hosts_by_serial.setdefault(serial, []).append(h["id"])
    raise FatalError(f"Fleet returned more than {MAX_PAGES} pages of hosts")


def coverage_end(apple, device_id):
    """Returns the latest end date (YYYY-MM-DD) across the device's non-canceled coverage, or None."""
    url = f"{APPLE_API_URL}/orgDevices/{urllib.parse.quote(device_id, safe='')}/appleCareCoverage"
    ends = []
    for page in apple.pages(url, "coverage"):
        for c in page.get("data") or []:
            attrs = c.get("attributes") or {}
            if not attrs.get("isCanceled") and attrs.get("endDateTime"):
                ends.append(attrs["endDateTime"][:10])
    return max(ends) if ends else None


def main():
    fleet_url = os.environ.get("FLEET_URL", "").strip().rstrip("/")  # e.g. https://fleet.example.com
    fleet_token = os.environ.get("FLEET_API_TOKEN", "").strip()
    assertion = os.environ.get("ABM_CLIENT_ASSERTION", "").strip()
    vital_name = os.environ.get("VITAL_NAME", "AppleCare warranty expiration")
    dry_run = os.environ.get("DRY_RUN") == "1"
    if not fleet_url or not fleet_token or not assertion:
        raise FatalError("Set FLEET_URL, FLEET_API_TOKEN, and ABM_CLIENT_ASSERTION")
    if not fleet_url.startswith("https://"):
        raise FatalError("FLEET_URL must start with https://, so the token isn't sent unencrypted")

    client_id, exp = assertion_claims(assertion)
    expires = time.strftime("%Y-%m-%d", time.gmtime(exp))
    if exp - time.time() < 3600:
        raise FatalError(f"The Apple Business client assertion expired on {expires}. Sign a new one and update ABM_CLIENT_ASSERTION.")
    days_left = int((exp - time.time()) // 86400)
    if days_left < ASSERTION_WARN_DAYS:
        print(f"Warning: the Apple Business client assertion expires on {expires} ({days_left} days). Sign a new one soon.", file=sys.stderr)

    apple = Apple(client_id, assertion)
    devices = apple_devices(apple)
    try:
        hosts_by_serial = fleet_hosts_by_serial(fleet_url, fleet_token)
    except SyncError as e:
        raise FatalError(f"Listing Fleet hosts failed: {e}")
    matched = sorted(s for s in devices if s in hosts_by_serial)
    print(f"{len(devices)} Apple Business devices, {len(matched)} in Fleet")

    updated = unchanged = 0
    failed = False
    for serial in matched:
        try:
            end = coverage_end(apple, devices[serial])
            if end is None:
                print(f"{serial}: no dated coverage, skipped")
                continue
            for host_id in hosts_by_serial[serial]:
                host = call("GET", f"{fleet_url}/api/v1/fleet/hosts/{host_id}?exclude_software=true", fleet_token)["host"]
                vital = next((v for v in host.get("custom_host_vitals") or [] if v.get("name") == vital_name), None)
                if vital is None:
                    raise FatalError(f"Custom host vital {vital_name!r} doesn't exist in Fleet")
                if vital.get("value") == end:
                    unchanged += 1
                    continue
                print(f"{host.get('display_name')} ({serial}): {vital.get('value') or '(empty)'} -> {end}")
                if not dry_run:
                    path = f"/api/v1/fleet/hosts/{host_id}/custom_host_vitals/{vital['custom_host_vital_id']}"
                    call("PUT", fleet_url + path, fleet_token, {"value": end})
                updated += 1
        except SyncError as e:
            # A missing permission fails every device the same way, so stop at the first one.
            if e.code in (401, 403):
                raise FatalError(f"{e} (check the Apple Business API account's role, or the Fleet API-only user's role and endpoints)")
            print(f"{serial}: {e}", file=sys.stderr)
            failed = True

    verb = "would update" if dry_run else "updated"
    print(f"Done: {verb} {updated} hosts, {unchanged} unchanged. Assertion expires {expires}.")
    return 1 if failed else 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (FatalError, SyncError) as e:
        print(f"Fatal: {e}", file=sys.stderr)
        sys.exit(2)
