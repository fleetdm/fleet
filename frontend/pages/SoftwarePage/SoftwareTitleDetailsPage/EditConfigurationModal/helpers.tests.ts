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

  it("returns error for malformed XML (unclosed tag)", () => {
    const error = validateXml("<dict><unclosed");
    expect(error).toBeTruthy();
    expect(typeof error).toBe("string");
  });

  it("returns error for malformed XML (mismatched tags)", () => {
    const error = validateXml("<dict><key>k</string></dict>");
    expect(error).toBeTruthy();
  });

  it("returns error when root element is not <dict>", () => {
    const error = validateXml("<array><string>hi</string></array>");
    expect(error).toMatch(/root element must be <dict>/i);
  });

  it("returns error when root element is <plist>", () => {
    const error = validateXml(
      "<plist><dict><key>k</key><string>v</string></dict></plist>"
    );
    expect(error).toMatch(/root element must be <dict>/i);
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
