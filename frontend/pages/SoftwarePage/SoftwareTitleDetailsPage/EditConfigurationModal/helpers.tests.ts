import {
  getErrorMessage,
  isErrorWithMessage,
  validateJson,
  validateXml,
  getPlatformLabel,
} from "./helpers";

describe("isErrorWithMessage", () => {
  it("returns true for an object with a message property", () => {
    expect(isErrorWithMessage({ message: "boom" })).toBe(true);
  });

  it("returns true for a native Error", () => {
    expect(isErrorWithMessage(new Error("native"))).toBe(true);
  });

  it("returns false for a plain string", () => {
    expect(isErrorWithMessage("oops")).toBe(false);
  });

  it("returns false for null", () => {
    expect(isErrorWithMessage(null)).toBe(false);
  });

  it("returns false for undefined", () => {
    expect(isErrorWithMessage(undefined)).toBe(false);
  });

  it("returns false for an object without message", () => {
    expect(isErrorWithMessage({ code: 42 })).toBe(false);
  });
});

describe("getErrorMessage", () => {
  const managedConfigErr = {
    response: {
      data: {
        errors: [{ name: "base", reason: "invalid managedConfiguration key" }],
      },
    },
  };

  const workProfileErr = {
    response: {
      data: {
        errors: [
          { name: "base", reason: "workProfileWidgets is not supported" },
        ],
      },
    },
  };

  it("returns Android-specific error for managedConfiguration on Android", () => {
    const result = getErrorMessage(managedConfigErr, false);
    expect(result).toBeTruthy();
    // Result is JSX, not a plain string
    expect(typeof result).not.toBe("string");
  });

  it("returns Android-specific error for workProfileWidgets on Android", () => {
    const result = getErrorMessage(workProfileErr, false);
    expect(result).toBeTruthy();
    expect(typeof result).not.toBe("string");
  });

  it("returns raw reason for managedConfiguration on iOS/iPadOS (not Android-specific message)", () => {
    const result = getErrorMessage(managedConfigErr, true);
    expect(result).toBe("invalid managedConfiguration key");
  });

  it("returns raw reason for workProfileWidgets on iOS/iPadOS", () => {
    const result = getErrorMessage(workProfileErr, true);
    expect(result).toBe("workProfileWidgets is not supported");
  });

  it("returns the reason string for a generic API error", () => {
    const err = {
      response: {
        data: {
          errors: [{ name: "base", reason: "something went wrong" }],
        },
      },
    };
    expect(getErrorMessage(err, false)).toBe("something went wrong");
    expect(getErrorMessage(err, true)).toBe("something went wrong");
  });

  it("returns default message when no reason can be extracted", () => {
    expect(getErrorMessage({}, false)).toBe(
      "Couldn't update configuration. Please try again."
    );
  });

  it("returns default message for null input", () => {
    expect(getErrorMessage(null, false)).toBe(
      "Couldn't update configuration. Please try again."
    );
  });

  it("returns 'doesn't exist' for an unknown $FLEET_VAR_ variable", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "configuration",
              reason: "unsupported variable $FLEET_VAR_BLA_BLA",
            },
          ],
        },
      },
    };
    expect(getErrorMessage(err, true)).toBe(
      `Couldn't add. Variable "$FLEET_VAR_BLA_BLA" doesn't exist.`
    );
  });

  it("returns 'doesn't exist' for an unknown $FLEET_VAR_ variable on Android", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "configuration",
              reason: "unsupported variable $FLEET_VAR_BLA_BLA",
            },
          ],
        },
      },
    };
    expect(getErrorMessage(err, false)).toBe(
      `Couldn't add. Variable "$FLEET_VAR_BLA_BLA" doesn't exist.`
    );
  });

  it.each([
    ["NDES_SCEP_CHALLENGE"],
    ["NDES_SCEP_PROXY_URL"],
    ["DIGICERT_DATA_myCA"],
    ["DIGICERT_PASSWORD_myCA"],
    ["SCEP_WINDOWS_CERTIFICATE_ID"],
    ["SMALLSTEP_SCEP_CHALLENGE_myCA"],
    ["SMALLSTEP_SCEP_PROXY_URL_myCA"],
    ["CUSTOM_SCEP_CHALLENGE_myCA"],
    ["CUSTOM_SCEP_PROXY_URL_myCA"],
    ["SCEP_RENEWAL_ID"],
  ])(
    "returns 'isn't supported in managed configuration' for %s",
    (varSuffix) => {
      const err = {
        response: {
          data: {
            errors: [
              {
                name: "configuration",
                reason: `unsupported variable $FLEET_VAR_${varSuffix}`,
              },
            ],
          },
        },
      };
      expect(getErrorMessage(err, true)).toBe(
        `Couldn't add. Variable "$FLEET_VAR_${varSuffix}" isn't supported in managed configuration. It can only be used in configuration profiles.`
      );
    }
  );

  it("returns 'doesn't exist' for a missing $FLEET_SECRET_ variable", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "configuration",
              reason:
                'Couldn\'t add. Secret variable "$FLEET_SECRET_BLA_BLA" missing from database',
            },
          ],
        },
      },
    };
    expect(getErrorMessage(err, true)).toBe(
      `Couldn't add. Variable "$FLEET_SECRET_BLA_BLA" doesn't exist.`
    );
  });

  it("returns 'doesn't exist' listing all variables for multiple missing $FLEET_SECRET_ variables", () => {
    const err = {
      response: {
        data: {
          errors: [
            {
              name: "configuration",
              reason:
                'Couldn\'t add. Secret variables "$FLEET_SECRET_A", "$FLEET_SECRET_B" missing from database',
            },
          ],
        },
      },
    };
    expect(getErrorMessage(err, true)).toBe(
      `Couldn't add. Variables "$FLEET_SECRET_A", "$FLEET_SECRET_B" doesn't exist.`
    );
  });
});

describe("validateJson", () => {
  it("returns null for valid JSON object", () => {
    expect(validateJson('{"key":"value"}')).toBeNull();
  });

  it("returns null for valid JSON array", () => {
    expect(validateJson("[1,2,3]")).toBeNull();
  });

  it("returns null for valid JSON string literal", () => {
    expect(validateJson('"hello"')).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(validateJson("")).toBeNull();
  });

  it("returns error message for malformed JSON", () => {
    const error = validateJson("{{ invalid");
    expect(error).toBeTruthy();
    expect(typeof error).toBe("string");
  });

  it("returns error message for trailing comma", () => {
    const error = validateJson('{"key":"value",}');
    expect(error).toBeTruthy();
  });

  it("returns error message for unquoted keys", () => {
    const error = validateJson("{key: true}");
    expect(error).toBeTruthy();
  });
});

describe("validateXml", () => {
  it("returns null for valid XML with <dict> root", () => {
    expect(
      validateXml("<dict><key>k</key><string>v</string></dict>")
    ).toBeNull();
  });

  it("returns null for empty string", () => {
    expect(validateXml("")).toBeNull();
  });

  it("returns null for multi-line XML with self-closing tags", () => {
    const xml = "<dict>\n\t<key>ForceLoginWithSSO</key>\n\t<true/>\n</dict>";
    expect(validateXml(xml)).toBeNull();
  });

  it("returns null for dict with multiple key-value pairs", () => {
    const xml = [
      "<dict>",
      "  <key>ForceLoginWithSSO</key>",
      "  <true/>",
      "  <key>SetSSOURL</key>",
      "  <string>https://example.com</string>",
      "</dict>",
    ].join("\n");
    expect(validateXml(xml)).toBeNull();
  });

  it("returns null for nested dict", () => {
    const xml = [
      "<dict>",
      "  <key>OuterKey</key>",
      "  <dict>",
      "    <key>InnerKey</key>",
      "    <string>value</string>",
      "  </dict>",
      "</dict>",
    ].join("\n");
    expect(validateXml(xml)).toBeNull();
  });

  it("returns 'Invalid XML' for malformed XML (unclosed tag)", () => {
    expect(validateXml("<dict><unclosed")).toBe("Invalid XML");
  });

  it("returns 'Invalid XML' for malformed XML (mismatched tags)", () => {
    expect(validateXml("<dict><key>k</string></dict>")).toBe("Invalid XML");
  });

  it("returns error when root element is not <dict>", () => {
    const error = validateXml("<array><string>hi</string></array>");
    expect(error).toMatch(/root element must be <dict>/i);
  });

  it("returns null for plist-wrapped dict", () => {
    expect(
      validateXml("<plist><dict><key>k</key><string>v</string></dict></plist>")
    ).toBeNull();
  });

  it("returns null for full plist with XML declaration and DOCTYPE", () => {
    const xml = [
      '<?xml version="1.0" encoding="UTF-8"?>',
      '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">',
      '<plist version="1.0">',
      "<dict>",
      "  <key>ForceLoginWithSSO</key>",
      "  <true/>",
      "</dict>",
      "</plist>",
    ].join("\n");
    expect(validateXml(xml)).toBeNull();
  });

  it("returns error when plist root value is not <dict>", () => {
    const error = validateXml(
      "<plist><array><string>hi</string></array></plist>"
    );
    expect(error).toMatch(/must contain a <dict>/i);
  });

  it("returns error when root element is <string>", () => {
    const error = validateXml("<string>just a string</string>");
    expect(error).toMatch(/root element must be <dict>/i);
  });

  it("returns null for empty dict", () => {
    expect(validateXml("<dict></dict>")).toBeNull();
  });

  it("returns null for self-closing dict", () => {
    expect(validateXml("<dict/>")).toBeNull();
  });
});

describe("getPlatformLabel", () => {
  it("returns iOS for ios", () => {
    expect(getPlatformLabel("ios")).toBe("iOS");
  });

  it("returns iPadOS for ipados", () => {
    expect(getPlatformLabel("ipados")).toBe("iPadOS");
  });

  it("returns Android for android", () => {
    expect(getPlatformLabel("android")).toBe("Android");
  });

  it("returns the raw string for unknown platforms", () => {
    expect(getPlatformLabel("darwin")).toBe("darwin");
  });

  it("returns the raw string for empty string", () => {
    expect(getPlatformLabel("")).toBe("");
  });
});
