import {
  generateAddCertAuthorityData,
  generateDropdownOptions,
  getDisplayErrMessage,
} from "./helpers";

/** Builds an API error in the shape getErrorReason reads. */
const apiError = (reason: string) => ({
  errors: [{ name: "base", reason }],
});

describe("AddCertAuthorityModal helpers", () => {
  describe("getDisplayErrMessage", () => {
    // The server names the CA type in the message ("Invalid NDES SCEP admin URL or
    // credentials"), so the match can't assume the reason starts with "invalid admin URL".
    it.each([
      "Couldn't add certificate authority. Invalid NDES SCEP admin URL or credentials. Please correct and try again.",
      "Couldn't edit certificate authority. Invalid NDES SCEP admin URL or credentials. Please correct and try again.",
    ])("returns the credentials error for: %s", (reason) => {
      expect(getDisplayErrMessage(apiError(reason))).toBe(
        "Invalid admin URL or credentials. Please correct and try again."
      );
    });

    it.each([
      [
        "Couldn't edit certificate authority. Invalid NDES SCEP admin URL. Please correct and try again.",
        "Invalid admin URL. Please correct and try again.",
      ],
      [
        "Couldn't edit certificate authority. Invalid NDES SCEP username. Please correct and try again.",
        "Invalid username. Please correct and try again.",
      ],
      [
        "Couldn't edit certificate authority. Invalid NDES SCEP password. Please correct and try again.",
        "Invalid password. Please correct and try again.",
      ],
      [
        "Couldn't edit certificate authority. Couldn't connect to NDES SCEP admin URL. Please correct and try again.",
        "Couldn't connect to admin URL. Please correct and try again.",
      ],
    ])("returns a specific error for: %s", (reason, expected) => {
      expect(getDisplayErrMessage(apiError(reason))).toBe(expected);
    });

    it.each([
      [
        "Couldn't add certificate authority. Invalid Hydrant URL. Please correct and try again.",
        "Invalid Hydrant URL. Please correct and try again.",
      ],
      [
        "Couldn't add certificate authority. Invalid DigiCert URL. Please correct and try again.",
        "Invalid DigiCert URL. Please correct and try again.",
      ],
      [
        "Couldn't add certificate authority. Invalid EST URL. Please correct and try again.",
        "Invalid EST URL. Please correct and try again.",
      ],
      [
        "Couldn't add certificate authority. Invalid NDES SCEP URL. Please correct and try again.",
        "Invalid NDES SCEP URL. Please correct and try again.",
      ],
      [
        "Couldn't add certificate authority. Invalid Smallstep SCEP URL. Please correct and try again.",
        "Invalid Smallstep SCEP URL. Please correct and try again.",
      ],
      [
        "Couldn't edit certificate authority. Invalid Hydrant URL. Please correct and try again.",
        "Invalid Hydrant URL. Please correct and try again.",
      ],
    ])("names the CA type in the URL error for: %s", (reason, expected) => {
      expect(getDisplayErrMessage(apiError(reason))).toBe(expected);
    });

    it("returns the SCEP URL error", () => {
      expect(
        getDisplayErrMessage(
          apiError(
            "Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again."
          )
        )
      ).toBe("Invalid SCEP URL. Please correct and try again.");
    });

    it("returns the generic URL error when the CA type isn't named", () => {
      expect(
        getDisplayErrMessage(
          apiError(
            'Couldn\'t add certificate authority. Post "https://example.com": dial tcp: lookup example.com: no such host'
          )
        )
      ).toBe("Invalid URL. Please correct and try again.");
    });

    it("returns the password cache error", () => {
      expect(
        getDisplayErrMessage(
          apiError(
            "Couldn't edit certificate authority. The NDES password cache is full. Please increase the number of cached passwords in NDES and try again."
          )
        )
      ).toBe(
        "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again."
      );
    });

    // These name a URL too, so they'd be captured by the CA-type URL match if it ran first.
    it.each([
      "Couldn't edit certificate authority. Invalid challenge URL or credentials. Please correct and try again.",
      "Couldn't edit certificate authority. Invalid Challenge URL. Please correct and try again.",
    ])("returns the challenge URL error for Smallstep: %s", (reason) => {
      expect(getDisplayErrMessage(apiError(reason))).toBe(
        "Invalid challenge URL or credentials. Please correct and try again."
      );
    });

    // The server's Google CAS errors name the field or Google setting to fix,
    // so they're shown as they are.
    it.each([
      [
        "Couldn't add certificate authority. The CA pool can only contain letters, numbers, hyphens and underscores.",
        "The CA pool can only contain letters, numbers, hyphens and underscores.",
      ],
      [
        "Couldn't add certificate authority. The service account key must be a JSON object.",
        "The service account key must be a JSON object.",
      ],
      [
        "Couldn't edit certificate authority. Couldn't authenticate with Google CAS. Check that the service account key is valid.",
        "Couldn't authenticate with Google CAS. Check that the service account key is valid.",
      ],
      [
        'Couldn\'t add certificate authority. CA pool "my-pool" wasn\'t found in project "my-project", location "us-central1".',
        'CA pool "my-pool" wasn\'t found in project "my-project", location "us-central1".',
      ],
      [
        "Couldn't edit certificate authority. Google couldn't issue an access token (HTTP 503). Try again later.",
        "Google couldn't issue an access token (HTTP 503). Try again later.",
      ],
      [
        'Couldn\'t add certificate authority. "GOOGLE_WIFI" name is already used by another Google CAS certificate authority. Please choose a different name and try again.',
        '"GOOGLE_WIFI" name is already used by another Google CAS certificate authority. Please choose a different name and try again.',
      ],
    ])("shows the Google CAS error as it is for: %s", (reason, expected) => {
      expect(getDisplayErrMessage(apiError(reason))).toBe(expected);
    });

    it("falls back to the default error for an unrecognized reason", () => {
      expect(getDisplayErrMessage(apiError("something unexpected"))).toBe(
        "Please try again."
      );
    });
  });
});

describe("generateDropdownOptions", () => {
  it("offers Google CAS after DigiCert", () => {
    const values = generateDropdownOptions(false).map((option) => option.value);
    expect(values.indexOf("google_cas")).toBe(values.indexOf("digicert") + 1);
  });
});

describe("generateAddCertAuthorityData", () => {
  it("maps the Google CAS form to the API's field names", () => {
    expect(
      generateAddCertAuthorityData("google_cas", {
        name: "GOOGLE_WIFI",
        projectId: "my-project",
        caPool: "my-ca-pool",
        location: "us-central1",
        apiKeyJson: '{"type":"service_account"}',
      })
    ).toEqual({
      google_cas: {
        name: "GOOGLE_WIFI",
        project: "my-project",
        ca_pool: "my-ca-pool",
        location: "us-central1",
        api_key_json: '{"type":"service_account"}',
      },
    });
  });
});
