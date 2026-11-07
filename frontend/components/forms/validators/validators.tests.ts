import {
  ValidPasswordErrorCode,
  isEqual,
  isPresent,
  validateYaml,
  isValidEmail,
  isValidHostname,
  validatePassword,
  isValidURL,
  isValidUuid,
} from "./validators";

describe("isPresent", () => {
  it("returns true for valid inputs", () => {
    [[1, 2, 3], { hello: "world" }, "hi@thegnar.co"].forEach((input) => {
      expect(isPresent(input)).toBe(true);
    });
  });

  it("returns false for empty / nullish inputs", () => {
    ["", undefined, false, null, "   ", "\t\n"].forEach((input) => {
      expect(isPresent(input)).toBe(false);
    });
  });
});

describe("isEqual (re-exported from lodash)", () => {
  it("returns true for equal inputs", () => {
    expect(isEqual("thegnarco", "thegnarco")).toBe(true);
    expect(isEqual(1, 1)).toBe(true);
    expect(isEqual(1.0, 1)).toBe(true);
    expect(isEqual(["thegnarco"], ["thegnarco"])).toBe(true);
    expect(isEqual({ hello: "world" }, { hello: "world" })).toBe(true);
    expect(isEqual({ foo: { bar: "baz" } }, { foo: { bar: "baz" } })).toBe(
      true
    );
  });

  it("returns false for unequal inputs", () => {
    expect(isEqual("thegnarco", "thegnar")).toBe(false);
    expect(isEqual(1, "thegnar")).toBe(false);
    expect(isEqual(["thegnarco"], [1])).toBe(false);
    expect(isEqual({ hello: "world" }, { hello: "foo" })).toBe(false);
    expect(isEqual({ foo: { bar: "baz" } }, { foo: { bar: "foo" } })).toBe(
      false
    );
  });
});

describe("isValidEmail", () => {
  it("returns true for valid emails", () => {
    ["hi@thegnar.co", "hi@gnar.dog", "fleet@gmail.com"].forEach((email) => {
      expect(isValidEmail(email)).toBe(true);
    });
  });

  it("returns false for invalid emails", () => {
    ["www.thegnar.co", "bill@shakespeare"].forEach((email) => {
      expect(isValidEmail(email)).toBe(false);
    });
  });
});

describe("isValidURL", () => {
  it("accepts http/https URLs without an explicit protocol list", () => {
    expect(isValidURL({ url: "https://fleetdm.com" })).toBe(true);
    expect(isValidURL({ url: "http://example.com/path?q=1" })).toBe(true);
  });

  it("enforces protocols when provided", () => {
    expect(
      isValidURL({ url: "ftp://example.com", protocols: ["http", "https"] })
    ).toBe(false);
    expect(
      isValidURL({ url: "https://example.com", protocols: ["http", "https"] })
    ).toBe(true);
  });

  it("allows localhost when opted in", () => {
    expect(isValidURL({ url: "http://localhost:8080" })).toBe(false);
    expect(
      isValidURL({ url: "http://localhost:8080", allowLocalHost: true })
    ).toBe(true);
  });
});

describe("isValidUuid", () => {
  it("accepts canonical UUIDs", () => {
    expect(isValidUuid("3d813cbb-47fb-32ba-91df-831e1593ac29")).toBe(true);
    expect(isValidUuid("00000000-0000-0000-0000-000000000000")).toBe(true);
  });

  it("rejects non-UUIDs", () => {
    ["not-a-uuid", "1234", "", "3d813cbb47fb32ba91df831e1593ac29"].forEach(
      (val) => {
        expect(isValidUuid(val)).toBe(false);
      }
    );
  });
});

describe("isValidHostname", () => {
  it("accepts FQDNs, IPs, localhost, and bracketed IPv6, with optional port", () => {
    type TestCase = [string, boolean];
    const testCases: TestCase[] = [
      ["fleet.example.com", true],
      ["fleet.example.com:8090", true],
      ["192.168.0.1", true],
      ["192.168.0.1:9090", true],
      ["2001:0db8:85a3:0000:0000:8a2e:0370:7334", true],
      ["[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080", true],
      ["localhost", true],
      ["localhost:3000", true],
      ["not a valid url!", false],
      ["example.com:", false],
      ["example.com:70000", false],
      ["256.256.256.256", false],
      ["2001:xyz:123", false],
      [":8080", false],
      ["[2001:db8::1]", false],
    ];
    testCases.forEach(([value, expected]) => {
      expect(isValidHostname(value)).toBe(expected);
    });
  });
});

describe("validatePassword", () => {
  it("is invalid when length / character-class requirements are not met", () => {
    const cases: {
      password: string;
      error: string;
      error_code: ValidPasswordErrorCode;
    }[] = [
      {
        password: "abc12!",
        error: "Password must be at least 12 characters",
        error_code: "too_short",
      },
      {
        password: "abc12456aaaa",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: "$%#12456!!!!",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: "password$%#xxx",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: "mypasswordxx",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: "123456789111",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password: "!@#$%^&*()!!!!",
        error: "Password must meet the criteria below",
        error_code: "invalid_format",
      },
      {
        password:
          "asasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasasas1!",
        error: "Password is over the character limit",
        error_code: "too_long",
      },
    ];
    cases.forEach(({ password, error, error_code }) => {
      expect(validatePassword(password)).toEqual({
        isValid: false,
        error,
        error_code,
      });
    });
  });

  it("is valid for 12-48 char passwords with a letter, number, and symbol", () => {
    [
      "p@assw0rd123",
      "This should be v4lid!",
      "admin123.pass",
      "pRZ'bW,6'6o}HnpL62",
    ].forEach((password) => {
      expect(validatePassword(password)).toEqual({
        isValid: true,
        error: "",
        error_code: "",
      });
    });
  });
});

describe("validateYaml", () => {
  const malformedYaml = [
    'key: "unterminated string',
    "key: value\n  bad_indent: true",
    "{ unbalanced: braces",
  ];

  const validYaml = [
    "spec:\n  config:\n    options:\n      logger_plugin: tls\n      pack_delimiter: /\n",
  ];

  it("rejects malformed yaml with a structured syntax error", () => {
    malformedYaml.forEach((y) => {
      const { error, isValid } = validateYaml(y);
      expect(isValid).toBe(false);
      if (typeof error === "string" || error === null) {
        throw new Error("expected a structured syntax error, got string|null");
      }
      expect(error.name).toBe("Syntax Error");
      expect(error.reason).toBeTruthy();
      expect(error.line).toBeGreaterThanOrEqual(0);
    });
  });

  it("rejects blank entries with a string error", () => {
    const { error, isValid } = validateYaml();
    expect(isValid).toBe(false);
    expect(error).toBe("YAML text must be present");
  });

  it("accepts valid yaml", () => {
    validYaml.forEach((y) => {
      const { error, isValid } = validateYaml(y);
      expect(isValid).toBe(true);
      expect(error).toBeNull();
    });
  });
});
