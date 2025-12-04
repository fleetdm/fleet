# Okta Dynamic SCEP Implementation - Engineering Handoff

**Branch:** `tux234-add-okta-scep`
**Status:** Functional Prototype (Tests Need Attention)
**Date:** 2025-12-03
**Contact:** Mitch

---

## Overview

This document describes a working implementation of Okta dynamic SCEP support for Fleet MDM. The feature is functional in staging (Render) with real devices, but needs test coverage and edge case hardening before production.

**What Works:**
- ✅ Full CA CRUD (create, read, update, delete Okta CAs)
- ✅ Challenge retrieval from Okta via HTTP Basic Auth
- ✅ Profile variable replacement (`$FLEET_VAR_OKTA_SCEP_CHALLENGE_*`, `$FLEET_VAR_OKTA_SCEP_PROXY_URL_*`)
- ✅ Device enrollment with Okta certificates
- ✅ Validation logic (ensures both URL and Challenge variables present)

**What Needs Work:**
- ❌ Test suite has failures (mocks, test cases need updates)
- ⚠️ Edge cases may not be fully covered
- ⚠️ Error handling could be more robust
- ⚠️ No integration tests for full enrollment flow

---

## Technical Decisions

### Decision 1: Model After Custom SCEP/Smallstep

**Why:** Okta supports multiple named CAs and dynamic challenges, exactly like Custom SCEP and Smallstep.

**Alternative Considered:** Model after NDES (single static CA)
**Rejected Because:** Okta customers may have multiple CAs per tenant.

**Implementation Pattern:**
```go
// Like Smallstep:
type OktaSCEPProxyCA struct {
    Name         string  // CA identifier
    URL          string  // SCEP endpoint
    ChallengeURL string  // Challenge retrieval endpoint
    Username     string  // HTTP Basic Auth
    Password     string  // HTTP Basic Auth
}

// Fleet variables:
$FLEET_VAR_OKTA_SCEP_CHALLENGE_<CA_NAME>
$FLEET_VAR_OKTA_SCEP_PROXY_URL_<CA_NAME>
```

### Decision 2: Reuse NDES Challenge Parsing

**Why:** Okta returns NDES-style HTML responses for challenges.

**Response Format:**
```html
<HTML><Body><P>The enrollment challenge password is: <B> ABC123XYZ789 </B></P></Body></HTML>
```

**Regex (Shared with NDES):**
```go
var oktaChallengeRegex = regexp.MustCompile(
    `(?i)The enrollment challenge password is: <B> (?P<password>\S*)`)
```

**Trade-off:** Brittle HTML parsing, but Okta maintains backward compatibility.

### Decision 3: Make Renewal ID Optional for Okta

**Why:** Okta explicitly doesn't support SCEP renewal.

**Okta Documentation:**
> "Okta as a CA doesn't support renewal requests. Instead, redistribute the profile before the certificate expires."

**Implementation:**
```go
type OktaVarsFound struct {
    urlCA          map[string]struct{}
    challengeCA    map[string]struct{}
    renewalIdFound bool
    supportsRenewal bool  // Always false for Okta
}

func (o *OktaVarsFound) Ok() bool {
    // Does NOT require renewalIdFound
    return len(o.urlCA) > 0 &&
           len(o.challengeCA) > 0 &&
           namesMatch(o.urlCA, o.challengeCA)
}
```

**Impact:** Profiles must be redistributed before expiration (typically 1 year).

### Decision 4: HTTP Basic Auth for Challenge Retrieval

**Why:** Okta's API uses HTTP Basic Auth (not OAuth/API keys).

**Security Considerations:**
- Passwords stored encrypted in database
- HTTPS enforced for all Okta communication
- Credentials validated during CA creation

**Alternative Considered:** OAuth 2.0
**Rejected Because:** Okta's SCEP challenge API uses Basic Auth (not our choice).

---

## Architecture

### Data Flow: Profile Deployment with Okta Variables

```
1. Admin creates profile with variables:
   $FLEET_VAR_OKTA_SCEP_CHALLENGE_OKTA_DT
   $FLEET_VAR_OKTA_SCEP_PROXY_URL_OKTA_DT

2. Device enrolls / profile deploys

3. Fleet detects variables during deployment
   ↓
4. Extract CA name: "OKTA_DT"
   ↓
5. Call GetOktaSCEPChallenge(ctx, oktaCA)
   ↓
6. HTTP GET https://okta.com/challenge
   Authorization: Basic <base64(username:password)>
   ↓
7. Parse HTML response
   ↓
8. Extract challenge: "ABC123XYZ789"
   ↓
9. Replace variables in profile:
   $FLEET_VAR_OKTA_SCEP_CHALLENGE_OKTA_DT → "ABC123XYZ789"
   $FLEET_VAR_OKTA_SCEP_PROXY_URL_OKTA_DT → "https://fleet.com/api/mdm/apple/scep/OKTA_DT"
   ↓
10. Deploy profile to device
    ↓
11. Device requests certificate via Fleet proxy
    ↓
12. Fleet proxies to Okta SCEP endpoint
    ↓
13. Certificate issued
```

### Key Files Modified

**Backend (Go):**
- `server/datastore/mysql/migrations/tables/20251203000000_AddOktaHostCertificateType.go` - Database migration
- `server/fleet/certificate_authorities.go` - OktaSCEPProxyCA type definition
- `server/fleet/mdm.go` - Fleet variable constants
- `ee/server/service/scep_proxy.go` - Challenge retrieval logic
- `ee/server/service/certificate_authorities.go` - CA CRUD and validation
- `server/service/apple_mdm.go` - Variable replacement during deployment
- `server/service/mdm_profiles.go` - Profile validation (OktaVarsFound)
- `server/mock/scep/config.go` - Mock implementations

**Frontend (TypeScript/React):**
- `frontend/pages/.../OktaForm/OktaForm.tsx` - CA configuration form
- `frontend/interfaces/certificates.ts` - Type definitions
- `frontend/interfaces/activity.ts` - Activity logging enums
- `frontend/pages/.../AddCertAuthorityModal/` - Add CA flow
- `frontend/pages/.../EditCertAuthorityModal/` - Edit CA flow

**Tests:**
- `ee/server/service/scep_proxy_test.go` - Unit tests for challenge retrieval
- `ee/server/service/certificate_authorities_test.go` - CA CRUD tests
- `ee/server/service/testing_utils.go` - Test helper (NewTestOktaChallengeServer)
- `ee/server/service/testdata/okta_challenge_response.html` - Test fixture

---

## Implementation Details

### Database Schema

**Migration:** `20251203000000_AddOktaHostCertificateType.go`

```sql
-- Add 'okta' to certificate_authorities.type
ALTER TABLE certificate_authorities
MODIFY COLUMN type ENUM(
    'digicert',
    'ndes_scep_proxy',
    'custom_scep_proxy',
    'hydrant',
    'smallstep',
    'custom_est_proxy',
    'okta'  -- NEW
) NOT NULL;

-- Add 'okta' to host_mdm_managed_certificates.type
ALTER TABLE host_mdm_managed_certificates
MODIFY COLUMN type ENUM(
    'digicert',
    'custom_scep_proxy',
    'ndes',
    'smallstep',
    'okta'  -- NEW
) NOT NULL DEFAULT 'ndes';
```

**Rollback Plan:** Remove 'okta' from enums (safe if no okta CAs created).

### Backend: Challenge Retrieval

**File:** `ee/server/service/scep_proxy.go`

```go
func (s *SCEPConfigService) GetOktaSCEPChallenge(
    ctx context.Context,
    ca fleet.OktaSCEPProxyCA,
) (string, error) {
    client := fleethttp.NewClient(fleethttp.WithTimeout(*s.Timeout))

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, ca.ChallengeURL, http.NoBody)
    if err != nil {
        return "", ctxerr.Wrap(ctx, err, "creating Okta challenge request")
    }

    req.SetBasicAuth(ca.Username, ca.Password)

    resp, err := client.Do(req)
    if err != nil {
        return "", ctxerr.Wrap(ctx, err, "sending Okta challenge request")
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", ctxerr.Errorf(ctx, "Okta challenge request failed with status %d", resp.StatusCode)
    }

    bodyText, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", ctxerr.Wrap(ctx, err, "reading Okta challenge response")
    }

    matches := oktaChallengeRegex.FindStringSubmatch(string(bodyText))
    if matches == nil {
        return "", ctxerr.New(ctx, "Okta SCEP challenge not found in response")
    }

    return matches[oktaChallengeRegex.SubexpIndex("password")], nil
}
```

**Error Handling:**
- ✅ HTTP errors (401, 403, 500)
- ✅ Malformed responses (regex mismatch)
- ✅ Network timeouts (configured via `s.Timeout`)
- ⚠️ May need retry logic for transient failures

**Testing:**
```go
// Test server in testing_utils.go
func NewTestOktaChallengeServer(t *testing.T) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify Basic Auth
        username, password, ok := r.BasicAuth()
        if !ok || username != "test-user" || password != "test-pass" {
            w.WriteHeader(http.StatusUnauthorized)
            return
        }

        // Return NDES-style HTML
        w.WriteHeader(http.StatusOK)
        w.Write(oktaChallengeResponse)  // From testdata/okta_challenge_response.html
    }))
}
```

### Backend: Profile Validation

**File:** `server/service/mdm_profiles.go`

**Purpose:** Ensure profiles have both URL and Challenge variables for the same CA name.

```go
type OktaVarsFound struct {
    urlCA          map[string]struct{}  // CA names with URL variable
    challengeCA    map[string]struct{}  // CA names with Challenge variable
    renewalIdFound bool                 // Optional for Okta
    supportsRenewal bool                // Always false
}

func (o *OktaVarsFound) Ok() bool {
    if o == nil {
        return true
    }
    if len(o.urlCA) == 0 || len(o.challengeCA) == 0 {
        return false
    }
    // Ensure URL and Challenge use same CA names
    for ca := range o.challengeCA {
        if _, ok := o.urlCA[ca]; !ok {
            return false
        }
    }
    return true  // Renewal ID not required
}
```

**Validation Examples:**

| Variables in Profile | Valid? | Error |
|---------------------|--------|-------|
| `CHALLENGE_CA1`, `URL_CA1` | ✅ Yes | - |
| `CHALLENGE_CA1`, `URL_CA2` | ❌ No | Name mismatch |
| `CHALLENGE_CA1` only | ❌ No | Missing URL |
| `CHALLENGE_CA1`, `URL_CA1`, `RENEWAL_ID` | ✅ Yes | Renewal ID optional |

### Frontend: Form Validation

**File:** `frontend/pages/.../OktaForm/helpers.ts`

```typescript
export const generateFormValidations = (
  oktaIntegrations: ICertificateAuthorityPartial[],
  isEditing: boolean
) => {
  return {
    name: {
      validations: [
        { name: "required", isValid: (d) => d.name.length > 0 },
        {
          name: "invalidCharacters",
          isValid: (d) => /^[a-zA-Z0-9_]+$/.test(d.name),
          message: "Only letters, numbers, underscores allowed"
        },
        {
          name: "unique",
          isValid: (d) => isEditing ||
            !oktaIntegrations.find(ca => ca.name === d.name),
          message: "Name already in use"
        }
      ]
    },
    scepURL: {
      validations: [
        { name: "required", isValid: (d) => d.scepURL.length > 0 },
        { name: "validURL", isValid: (d) => isValidUrl(d.scepURL) }
      ]
    },
    challengeURL: {
      validations: [
        { name: "required", isValid: (d) => d.challengeURL.length > 0 },
        { name: "validURL", isValid: (d) => isValidUrl(d.challengeURL) }
      ]
    },
    username: {
      validations: [
        { name: "required", isValid: (d) => d.username.length > 0 }
      ]
    },
    password: {
      validations: [
        { name: "required", isValid: (d) => d.password.length > 0 }
      ]
    }
  };
};
```

**API Validation (Backend):**
When CA is created, Fleet calls:
1. `ValidateSCEPURL()` - Ensures SCEP endpoint is reachable
2. `ValidateOktaChallengeURL()` - Actually retrieves a challenge to verify credentials

This catches configuration errors before saving.

---

## Known Issues & Limitations

### 1. Okta Doesn't Support Renewal

**Impact:** Certificates must be reissued before expiration.

**Recommendation:**
- Monitor certificate expiration dates
- Set up alerts 30 days before expiration
- Redistribute profiles to trigger reissuance
- Consider automation for large fleets

**Code Location:** `server/service/mdm_profiles.go:366-395` (OktaVarsFound)

### 2. Challenge TTL: 60 Minutes

**Impact:** Device must complete enrollment within 60 minutes of profile deployment.

**Edge Cases:**
- Device offline when profile deployed → Challenge expires
- Network issues during enrollment → Challenge wasted
- Multiple deployment attempts → Each consumes a new challenge

**Recommendation:**
- Monitor enrollment failures
- Implement retry logic with new challenge retrieval
- Alert on high failure rates

### 3. HTML Parsing Brittleness

**Current Implementation:** Regex on HTML response from Okta.

**Risk:** If Okta changes HTML format, parsing breaks.

**Mitigation:**
- Log full response body on parse failures
- Consider asking Okta for JSON API (long-term)
- Add test fixture for any new HTML variations

**Code Location:** `ee/server/service/scep_proxy.go` (oktaChallengeRegex)

### 4. No Retry Logic for Challenge Retrieval

**Current Behavior:** Single HTTP request, no retries.

**Impact:** Transient network errors cause enrollment failures.

**Recommendation:**
- Add exponential backoff retry (e.g., 3 attempts)
- Only retry on 5xx errors (not 401/403)
- Respect Okta rate limits

### 5. Test Coverage Gaps

**Current State:**
- ✅ Unit tests for challenge retrieval (happy path)
- ✅ Unit tests for validation logic
- ❌ Integration tests (full enrollment flow)
- ❌ Mock functions incomplete (some tests fail)
- ❌ Edge case tests (malformed HTML, timeouts, etc.)

---

## Test Failures - Action Items

### Step 1: Run Tests and Capture Output

```bash
cd .worktrees/okta-scep
go test ./ee/server/service/... -v 2>&1 | tee test-output.txt
go test ./server/service/... -v 2>&1 | tee -a test-output.txt
```

### Step 2: Likely Issues

**Mock Functions:**
- `server/mock/scep/config.go` has Okta methods, but may need default implementations
- Tests expecting nil mocks will panic

**Test Cases:**
- `ee/server/service/certificate_authorities_test.go` needs Okta test scenarios
- `server/service/mdm_profiles_test.go` needs Okta variable validation tests

**Database State:**
- Integration tests may need migration applied
- Test fixtures may not include Okta CAs

### Step 3: Recommended Fixes

**Add Okta Test Cases:**
```go
// In certificate_authorities_test.go
{
    name: "okta happy path",
    user: adminUser,
    ca: fleet.CertificateAuthorityPayload{
        Okta: &fleet.OktaSCEPProxyCA{
            Name:         "test-okta",
            URL:          scepServer.URL + "/scep",
            ChallengeURL: oktaChallengeServer.URL,
            Username:     "test-user",
            Password:     "test-pass",
        },
    },
},
```

**Update Mocks:**
```go
// In server/mock/scep/config.go - ensure default implementations
func (s *SCEPConfigService) ValidateOktaChallengeURL(
    ctx context.Context,
    ca fleet.OktaSCEPProxyCA,
) error {
    s.mu.Lock()
    s.ValidateOktaChallengeURLFuncInvoked = true
    s.mu.Unlock()
    if s.ValidateOktaChallengeURLFunc != nil {
        return s.ValidateOktaChallengeURLFunc(ctx, ca)
    }
    return nil  // Default: no error
}
```

**Add Integration Test:**
```go
func TestOktaSCEPEndToEnd(t *testing.T) {
    // 1. Create Okta CA
    // 2. Create profile with variables
    // 3. Mock device enrollment
    // 4. Verify challenge retrieved
    // 5. Verify variables replaced
    // 6. Verify certificate issued
}
```

---

## Edge Cases to Consider

### 1. Multiple Okta CAs in Same Profile

**Scenario:** Profile has variables for two Okta CAs:
```xml
$FLEET_VAR_OKTA_SCEP_CHALLENGE_CA1
$FLEET_VAR_OKTA_SCEP_CHALLENGE_CA2
```

**Expected Behavior:** Both challenges retrieved, both variables replaced.

**Test Status:** ⚠️ Not tested

### 2. CA Deleted While Profile Active

**Scenario:**
1. Profile deployed with `$FLEET_VAR_OKTA_SCEP_CHALLENGE_CA1`
2. Admin deletes `CA1` from Fleet
3. Profile redeployed

**Current Behavior:** Likely error (CA not found).

**Recommendation:** Prevent CA deletion if used in active profiles.

### 3. Okta API Rate Limiting

**Scenario:** Large fleet (1000s of devices) enrolling simultaneously.

**Risk:** Exceed Okta challenge API rate limits.

**Recommendation:**
- Document rate limits
- Implement request throttling
- Consider challenge caching (if Okta allows reuse)

### 4. Challenge Retrieval Timeout

**Current Timeout:** Configured via `SCEPConfigService.Timeout`

**Scenario:** Okta API slow/unavailable.

**Recommendation:**
- Set reasonable timeout (e.g., 30s)
- Log timeout errors
- Alert on high timeout rates

### 5. Unicode in Challenge Passwords

**Current Implementation:** Regex expects `\S*` (non-whitespace).

**Risk:** If Okta returns unicode characters, may fail.

**Test Status:** ⚠️ Not tested with unicode

---

## Production Readiness Checklist

### Code Quality
- [ ] All tests pass
- [ ] Integration test for full enrollment flow
- [ ] Edge case tests (timeouts, malformed responses, unicode)
- [ ] Linting passes: `golangci-lint run`
- [ ] Frontend linting: `npm run lint`

### Documentation
- [ ] API documentation updated (new endpoints)
- [ ] User guide: How to configure Okta CA
- [ ] Mobile config examples
- [ ] Known limitations documented (no renewal)
- [ ] Troubleshooting guide (common errors)

### Security
- [ ] Credentials encrypted at rest (verify)
- [ ] HTTPS enforced for Okta communication (verify)
- [ ] Rate limiting on CA creation (prevent brute force)
- [ ] Audit logging for CA operations

### Monitoring
- [ ] Metrics: Challenge retrieval success/failure rate
- [ ] Metrics: SCEP enrollment success/failure rate
- [ ] Alerts: High failure rates
- [ ] Alerts: Okta API errors
- [ ] Dashboard: Certificate expiration dates

### Operations
- [ ] Database migration tested (up and down)
- [ ] Rollback plan documented
- [ ] Performance tested (large fleet scenario)
- [ ] Load tested (concurrent enrollments)

### Customer Success
- [ ] Release notes written
- [ ] Migration guide for existing CAs (if applicable)
- [ ] Support team trained
- [ ] Known issues documented

---

## Example Configuration

### Okta CA Setup (UI)

```
Name: OKTA_DT
SCEP URL: https://your-tenant.okta.com/scep/v1/MDAbc123
Challenge URL: https://your-tenant.okta.com/scep/v1/challenge/MDAbc123
Username: scep-admin
Password: ••••••••
```

### Mobile Config Profile

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
          "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadContent</key>
            <dict>
                <key>Challenge</key>
                <string>$FLEET_VAR_OKTA_SCEP_CHALLENGE_OKTA_DT</string>

                <key>URL</key>
                <string>$FLEET_VAR_OKTA_SCEP_PROXY_URL_OKTA_DT</string>

                <key>Key Type</key>
                <string>RSA</string>

                <key>Key Usage</key>
                <integer>5</integer>

                <key>Keysize</key>
                <integer>2048</integer>

                <key>Subject</key>
                <array>
                    <array>
                        <array>
                            <string>CN</string>
                            <string>%SerialNumber%</string>
                        </array>
                    </array>
                    <array>
                        <array>
                            <string>OU</string>
                            <string>FLEET DEVICE MANAGEMENT</string>
                        </array>
                    </array>
                </array>
            </dict>
            <key>PayloadType</key>
            <string>com.apple.security.scep</string>
            <key>PayloadUUID</key>
            <string>GENERATE-UNIQUE-UUID</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>GENERATE-UNIQUE-UUID</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

**Key Differences from Other SCEP CAs:**
- ❌ No `$FLEET_VAR_SCEP_RENEWAL_ID` in Subject CN (Okta doesn't support renewal)
- ✅ Both Challenge and URL variables required
- ✅ Variables include CA name suffix (e.g., `_OKTA_DT`)

---

## Questions for Engineering Team

1. **Test Strategy:** What's the preferred approach for integration tests? Mock Okta API or use test fixtures?

2. **Rate Limiting:** Should we implement client-side throttling for Okta challenge requests?

3. **Challenge Caching:** Can Okta challenges be reused for multiple devices, or strictly one-per-device?

4. **Error Handling:** Current implementation logs errors but doesn't expose detailed error messages to UI. Desired behavior?

5. **Monitoring:** What metrics/alerts are most important for Okta SCEP operations?

6. **Rollout Plan:** Gradual rollout (feature flag) or all-at-once?

7. **Migration:** Any existing customers using Okta certificates via workarounds that need migration?

---

## References

**Okta Documentation:**
- [Dynamic SCEP for macOS with Jamf](https://help.okta.com/oie/en-us/content/topics/identity-engine/devices/okta-ca-dynamic-scep-macos-jamf.htm)

**Fleet Code Patterns:**
- NDES: `ee/server/service/scep_proxy.go` (challenge HTML parsing)
- Custom SCEP: `server/service/mdm_profiles.go` (variable validation)
- Smallstep: `server/fleet/certificate_authorities.go` (named CA pattern)

**Testing Resources:**
- Test server: `ee/server/service/testing_utils.go:NewTestOktaChallengeServer`
- Test fixture: `ee/server/service/testdata/okta_challenge_response.html`
- Existing SCEP tests: `ee/server/service/scep_proxy_test.go`

---

## Commit History

**Branch:** `tux234-add-okta-scep`

1. `f114dc8` - Backend: Add Okta SCEP CA support (database, API, validation)
2. `c4bb40f` - Frontend: Add Okta SCEP UI (forms, dropdowns, activity)
3. `bfc1bd9` - Fix: Add Okta variable validation
4. `d3cec80` - Fix: Make renewal ID optional (Okta limitation)

---

## Next Steps

1. **Review this document** - Validate technical decisions and approach
2. **Run test suite** - Identify specific test failures
3. **Prioritize fixes** - Tests, edge cases, or documentation?
4. **Assign ownership** - Who takes point on productionization?
5. **Set timeline** - What's realistic for production-ready?

**Happy to collaborate on any of the above!** This prototype provides a foundation, but production readiness requires your team's expertise.
