# Tasks

Start from the PoC branch (`origin/add-ejbca-rest-ca-poc`) where it already matches the spec, and keep each group's tests and docs in the same group. A local EJBCA for verification is described in `ai/ejbca/live-poc-findings.md`.

## 1. Types, migration, and datastore

- [ ] 1.1 Add `CATypeEJBCA`, the `EJBCACA` type, payload and update-payload types with `Preprocess`, `Equals`, and validation helpers in `server/fleet/certificate_authorities.go`; verify with unit tests in `server/fleet/certificate_authorities_test.go` covering required fields, name rules, and masked values.
- [ ] 1.2 Add a migration (use `/new-migration`) for the `ejbca` CA type, the EJBCA columns, the `ejbca` asset type, and the `ejbca` managed-certificate type; verify with a migration test asserting enum values and columns, and run `make migration` ordering checks.
- [ ] 1.3 Implement datastore create, read (masked and unmasked), update, delete, and batch apply for EJBCA in `server/datastore/mysql/certificate_authorities.go`, storing the key encrypted; verify with `MYSQL_TEST=1 go test ./server/datastore/mysql/ -run CertificateAuthorit` covering secrets never returned masked-off by default.
- [ ] 1.4 Add the resend-on-change rule for EJBCA certificate-affecting fields (username template, UPNs, CA name, certificate profile, end entity profile) in the same transaction as the update, mirroring the DigiCert helper; verify with datastore tests where changing a certificate field resends and changing URL, client certificate, or trust bundle does not.

## 2. EJBCA client

- [ ] 2.1 Port `ee/server/service/ejbca/` from the PoC with `fleet.EJBCAService` wiring, replacing `GET /v1/ca/status` with an authenticated `GET /v1/ca` check that also confirms the configured CA name is visible; verify with unit tests on `httptest` TLS servers (accepted cert, 403, handshake failure, CA name missing, protocol disabled).
- [ ] 2.2 Fix the UPN SAN encoding to `[0] { OID, [0] EXPLICIT UTF8String }` and verify with a test that parses the produced extension with an independent ASN.1 parse and checks one and several UPNs.
- [ ] 2.3 Replace the 404 and 422 branches in error mapping with EJBCA's real 400 and 403 behavior and EJBCA's `error_message` text (see spec "Enrollment failure reporting"); verify with table-driven tests built from the recorded responses in `ai/ejbca/live-poc-findings.md`.
- [ ] 2.4 Replace the `openssl` subprocess with a pure-Go BER to DER normalizer in front of the PKCS#12 decode; verify with fixtures exported from EJBCA RA Web (BER) and the CLI batch tool, a DER fixture, wrong-password and malformed-length cases, and confirm no `os/exec` import remains.
- [ ] 2.5 Run the client against a live local EJBCA (container per the findings notes, role with only the documented rules) for new enrollment, same-username re-enrollment, and UPN; verify with a documented manual run, keeping the live test skipped unless an env var is set.

## 3. Service, API, and activities

- [ ] 3.1 Add EJBCA validation, create, update, and delete in `ee/server/service/certificate_authorities.go` including Premium gating, the connection check, P12 decode, and variable allow-list validation for the username template and UPNs; verify with `go test ./ee/server/service/ -run CertificateAuthorit` for each spec scenario.
- [ ] 3.2 Add `added_ejbca`, `edited_ejbca`, and `deleted_ejbca` activities in `server/fleet/activities.go` and the activity docs generator; verify with the activity tests and by running `make generate-doc` (or the repo's audit-log generation) with no diff in `docs/Contributing/reference/audit-logs.md` after updating it.
- [ ] 3.3 Wire the EJBCA service in `cmd/fleet/serve.go`, `server/service/service.go`, and `server/service/svctest/service.go`; verify `go build ./...` and `go test ./server/service/` pass (uninitialized mocks crash otherwise).
- [ ] 3.4 Add integration tests in `server/service/integration_certificate_authorities_test.go` for create, list masking, update, delete, duplicate name, Free tier rejection, and bad credentials; verify with `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/ -run CertificateAuthorities`.

## 4. Profile variables and delivery

- [ ] 4.1 Add `EJBCA_DATA_` and `EJBCA_PASSWORD_` variable prefixes and the EJBCA allow-list in `server/fleet/mdm.go` and `server/variables/variables.go`, and the pair, CA-exists, and pkcs12-placement validation in `server/service/mdm_profiles.go` (and Windows rejection in `windows_mdm_profiles.go`); verify with tests for valid profile, missing pair, unknown CA, and wrong payload.
- [ ] 4.2 Extend `server/mdm/apple/profile_processor.go` to enroll per host, expand username and UPN variables, substitute the data and password variables, write the managed-certificate record, and report failures with the spec messages; verify with profile processor tests including a failed enrollment detail.
- [ ] 4.3 Resend EJBCA profiles when a host's IdP values change and the CA fields use them, mirroring #52368; verify with a datastore test for an IdP email change.
- [ ] 4.4 Add an end-to-end check on a macOS host with a Wi-Fi style pkcs12 profile against the local EJBCA (new host, resend, certificate-field edit, client certificate rotation); verify the certificate appears in the host keychain and in the host's Fleet-managed certificate list.

## 5. GitOps and fleetctl

- [ ] 5.1 Add `ejbca` to the grouped certificate authorities spec, parsing, and batch apply in `server/fleet/certificate_authorities.go`, `ee/server/service/certificate_authorities.go`, and `server/service/client.go`; verify with GitOps unit tests for add, change, remove, and invalid input.
- [ ] 5.2 Emit EJBCA in `fleetctl generate-gitops` with secret placeholders in `cmd/fleetctl/fleetctl/generate_gitops.go`; verify by updating the golden files under `cmd/fleetctl/fleetctl/testdata/generateGitops/` and running `go test ./cmd/fleetctl/fleetctl/ -run GenerateGitops`.
- [ ] 5.3 Run `fleetctl gitops` against a local server for create, update, and delete of an EJBCA CA; verify activities are recorded and the CA matches the YAML.

## 6. Frontend

- [ ] 6.1 Add the EJBCA form (PKCS#12 upload with password, URL, trust bundle, EJBCA CA, certificate profile, end entity profile, username template, UPNs) and add and edit modal wiring following the DigiCert form and the Figma in issue #30986; verify with Jest tests for validation and payload mapping (`yarn test`).
- [ ] 6.2 Add the EJBCA activity feed entries, interfaces, and API service changes in `frontend/interfaces/` and `frontend/services/entities/certificates.ts`; verify with `GlobalActivityItem` tests for all three activities.
- [ ] 6.3 Show the client certificate expiry on the CA list with a warning style under 30 days and an error style under 7 days; verify with component tests for each threshold.
- [ ] 6.4 Add EJBCA to the command palette entries only if DigiCert has an equivalent entry; verify with the palette tests or note none exists.

## 7. Docs, changes file, and finish

- [ ] 7.1 Add the `changes/` file (no extension) for the feature and the DigiCert-aligned reissue behavior; verify the file exists and follows the format of `changes/52368-digicert-fleet-variables`.
- [ ] 7.2 Update the contributor dev guide `docs/Contributing/product-groups/security-compliance/ejbca-rest-testing.md` using `ai/ejbca/live-poc-findings.md`: `localhost` setup without `/etc/hosts`, auto-created end entities, the minimal role rules, `/v1/ca` verification, and removal of the extension override claim; verify by following the guide on a clean container.
- [ ] 7.3 In a separate docs PR off the docs branch, extend the REST API, YAML, audit log, fleet variables, and Wi-Fi guide docs with the full field list and EJBCA-side prerequisites; verify the documented request and YAML work as written against a local server.
- [ ] 7.4 Update the issue #30986 test plan to the confirmed reissue rule and fill its edge cases (bad credentials, unknown profile names, missing role rules, repeat enrollment); verify the plan matches the spec scenarios.
- [ ] 7.5 Run `make lint-go-incremental`, `make lint-js`, and the affected test packages, then `/go-reviewer` over the diff; verify all pass.
