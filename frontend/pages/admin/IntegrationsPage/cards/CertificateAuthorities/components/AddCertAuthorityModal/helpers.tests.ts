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

    it("returns the SCEP URL error", () => {
      expect(
        getDisplayErrMessage(
          apiError(
            "Couldn't edit certificate authority. Invalid SCEP URL. Please correct and try again."
          )
        )
      ).toBe("Invalid SCEP URL. Please correct and try again.");
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

    it("returns the challenge URL error for Smallstep", () => {
      expect(
        getDisplayErrMessage(
          apiError(
            "Couldn't edit certificate authority. Invalid challenge URL or credentials. Please correct and try again."
          )
        )
      ).toBe(
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
