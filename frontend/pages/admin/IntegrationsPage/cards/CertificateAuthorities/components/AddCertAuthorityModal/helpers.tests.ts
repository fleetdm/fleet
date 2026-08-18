import { getDisplayErrMessage } from "./helpers";

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

    it("falls back to the default error for an unrecognized reason", () => {
      expect(getDisplayErrMessage(apiError("something unexpected"))).toBe(
        "Please try again."
      );
    });
  });
});
