# expand-secrets

Expands a single "bundle" secret — a JSON object like
`{"SECRET_NAME": "value", ...}` — into individual masked environment
variables for the rest of the job. This lets us consolidate groups of
related secrets into one repository secret and stay under GitHub's
100-secret-per-repo limit (bundle values can be up to 48KB).

## Usage in a workflow

```yaml
- name: Checkout
  uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1 # required before using a local action

- name: Expand secrets
  uses: ./.github/actions/expand-secrets
  with:
    secrets-json: ${{ secrets.DOGFOOD_GITOPS_SECRETS_JSON }}

# Later steps reference env.SECRET_NAME instead of secrets.SECRET_NAME.
- name: Do something
  env:
    MY_TOKEN: ${{ env.SOME_TOKEN_FROM_THE_BUNDLE }}
  run: ...
```

Every value is re-masked line-by-line with `::add-mask::`, so extracted
values (including multi-line certificates and XML) never appear in logs.

## Building a bundle

Secret values can't be read back out of GitHub, so build the JSON from
your source of truth (e.g. 1Password). One easy path: create a directory
with one file per secret (filename = secret name, file content = value),
then:

```bash
cd my-secrets-dir
for f in *; do
  jq -n --arg k "$f" --rawfile v "$f" '{($k): ($v | rtrimstr("\n"))}'
done | jq -s 'add' | gh secret set DOGFOOD_GITOPS_SECRETS_JSON --repo fleetdm/fleet
```

`rtrimstr("\n")` strips the single trailing newline most editors add;
exact values matter for enroll secrets and API tokens.

## Rotating one value

Rebuild the whole bundle from the source of truth (the loop above), or
patch a single key if you have the current JSON locally:

```bash
jq --arg v "$NEW_VALUE" '.SECRET_NAME = $v' bundle.json |
  gh secret set DOGFOOD_GITOPS_SECRETS_JSON --repo fleetdm/fleet
```

## Caveats

- Any job that expands a bundle gets access to **all** keys in it. Keep
  bundles scoped to one workflow/purpose rather than one mega-bundle.
- Secrets (and therefore bundles) are unavailable to pull requests from
  forks; the action logs a notice and exports nothing in that case.
- Values must be JSON strings. Booleans/numbers should be quoted when
  building the bundle.
