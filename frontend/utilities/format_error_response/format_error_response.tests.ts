import formatErrorResponse from "./index";

describe("formatErrorResponse - utility", () => {
  it("surfaces the global reason as `base` and copies the http status", () => {
    const result = formatErrorResponse({
      status: 500,
      message: { errors: [{ name: "", reason: "boom" }] },
    });
    expect(result.base).toBe("boom");
    expect(result.http_status).toBe(500);
  });

  it("also keys messages by field name so forms can render them inline", () => {
    const result = formatErrorResponse({
      status: 422,
      message: {
        errors: [
          { name: "email", reason: "Email already exists" },
          { name: "name", reason: "Name cannot be empty" },
        ],
      },
    });
    expect(result.email).toBe("Email already exists");
    expect(result.name).toBe("Name cannot be empty");
    // Legacy base behavior: last non-"base" reason wins.
    expect(result.base).toBe("Name cannot be empty");
  });

  it("joins repeated field errors with a comma", () => {
    const result = formatErrorResponse({
      status: 422,
      message: {
        errors: [
          { name: "email", reason: "already exists" },
          { name: "email", reason: "is not valid" },
        ],
      },
    });
    expect(result.email).toBe("already exists, is not valid");
  });

  it("lets an unnamed error after a named one overwrite `base`", () => {
    const result = formatErrorResponse({
      status: 422,
      message: {
        errors: [
          { name: "email", reason: "already exists" },
          { name: "", reason: "something else went wrong" },
        ],
      },
    });
    expect(result.base).toBe("something else went wrong");
    expect(result.email).toBe("already exists");
  });

  it("joins repeated explicit `base` errors with a comma", () => {
    const result = formatErrorResponse({
      status: 500,
      message: {
        errors: [
          { name: "base", reason: "something broke" },
          { name: "base", reason: "and another thing" },
        ],
      },
    });
    expect(result.base).toBe("something broke, and another thing");
  });

  it("returns only the http status when there are no errors", () => {
    const result = formatErrorResponse({ status: 500 });
    expect(result).toEqual({ http_status: 500 });
  });
});
