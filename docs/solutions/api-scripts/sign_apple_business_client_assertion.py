#!/usr/bin/env python3
"""Sign a client assertion for an Apple Business API account.

The assertion is a JWT signed with the API account's private key. Apple accepts it
for up to 180 days, so a scheduled sync only needs the assertion, never the key.
Used by sync_applecare_coverage_to_fleet.py.

Setup guide: https://fleetdm.com/guides/sync-applecare-coverage-from-apple-business

Usage
-----
  python3 sign_apple_business_client_assertion.py --key KEY.pem \\
      --client-id BUSINESSAPI.<uuid> --key-id <uuid>

Prints the assertion to stdout and its expiry date to stderr. Store the assertion
like a password: until it expires, it can get access tokens for the API account.

Requires the openssl command. No third-party packages.
"""
import argparse, base64, json, subprocess, sys, time, uuid

AUDIENCE = "https://account.apple.com/auth/oauth2/v2/token"
LIFETIME_DAYS = 179  # Apple rejects assertions that expire 180 or more days after they're issued.


def b64url(data):
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode()


def der_to_raw(der):
    """Converts openssl's DER ECDSA signature to the 64-byte r||s form that JWTs use."""
    if len(der) < 8 or der[0] != 0x30 or der[1] != len(der) - 2:
        raise ValueError("unexpected signature encoding")
    out, i = b"", 2
    for _ in range(2):
        if der[i] != 0x02:
            raise ValueError("unexpected signature encoding")
        n = der[i + 1]
        value = der[i + 2:i + 2 + n].lstrip(b"\x00")
        if len(value) > 32:
            raise ValueError("signature is not P-256")
        out += value.rjust(32, b"\x00")
        i += 2 + n
    return out


def main():
    p = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    p.add_argument("--key", required=True, help="private key .pem downloaded from Apple Business")
    p.add_argument("--client-id", required=True, help="Client ID, like BUSINESSAPI.<uuid>")
    p.add_argument("--key-id", required=True, help="Key ID shown with the Client ID")
    args = p.parse_args()

    key_text = subprocess.run(["openssl", "pkey", "-in", args.key, "-noout", "-text"],
                              capture_output=True, text=True, check=True).stdout
    if "prime256v1" not in key_text and "P-256" not in key_text:
        sys.exit(f"{args.key} is not an EC P-256 key")

    now = int(time.time())
    header = {"alg": "ES256", "kid": args.key_id}
    payload = {"iss": args.client_id, "sub": args.client_id, "aud": AUDIENCE,
               "iat": now, "exp": now + LIFETIME_DAYS * 86400, "jti": str(uuid.uuid4())}
    signing_input = "%s.%s" % (b64url(json.dumps(header, separators=(",", ":")).encode()),
                               b64url(json.dumps(payload, separators=(",", ":")).encode()))
    der = subprocess.run(["openssl", "dgst", "-sha256", "-sign", args.key],
                         input=signing_input.encode(), capture_output=True, check=True).stdout
    print(f"{signing_input}.{b64url(der_to_raw(der))}")
    print(f"Expires {time.strftime('%Y-%m-%d', time.gmtime(payload['exp']))}. Sign a new one before then.", file=sys.stderr)


if __name__ == "__main__":
    main()
