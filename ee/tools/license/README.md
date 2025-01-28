# Fleet license key generation

This directory contains the Node script for generating Fleet license keys.

## Usage

### Setup

Install the JS dependencies:

```sh
yarn
```

Put the private key and passphrase into files (avoid using shell commands to prevent secrets from leaking into shell history). They are in the "Licensing" Vault in 1Password. Speak to @lukeheath or @mikermcneil if you need to get access to these secrets.

```sh
nano key.pem
nano passphrase.txt
```

### Generate a key

Run `./license.js generate` to generate a key. For example:

```sh
./license.js generate --private-key key.pem --key-passphrase passphrase.txt --expiration 2022-01-01 --customer test --devices 100 --note 'for development only'
eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjQwOTk1MjAwLCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMCwibm90ZSI6ImZvciBkZXZlbG9wbWVudCBvbmx5IiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjIyNDI2NTg2fQ.WmZ0kG4seW3IrNvULCHUPBSfFdqj38A_eiXdV_DFunMHechjHbkwtfkf1J6JQJoDyqn8raXpgbdhafDwv3rmDw
```

See `./license.js generate --help` for more details on arguments.

## Key format

License keys are JWTs signed with the `ES256` method.

Check the contents of a key with [https://jwt.io/](https://jwt.io/).

The key generated above contains the following payload:

```json
{
  "iss": "Fleet Device Management Inc.",
  "exp": 1640995200,
  "sub": "development",
  "devices": 100,
  "note": "for development only",
  "tier": "premium",
  "iat": 1622426586
}
```

- `devices` refers to the number of licensed devices.
- `note` includes any additional note about the terms of the license.
- `tier` is the license Tier: currently `premium` = paid, `free` = free (though we do not plan to issue license keys for `free`)
- Other claims use the meanings [described in the JWT specification](https://datatracker.ietf.org/doc/html/rfc7519#section-4.1).
