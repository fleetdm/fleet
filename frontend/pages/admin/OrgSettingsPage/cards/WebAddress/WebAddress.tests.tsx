import { trimFormData } from "hooks/useFormValidation";

import { validate } from "./WebAddress";

describe("WebAddress validate", () => {
  it("accepts a valid URL with surrounding whitespace", () => {
    expect(validate({ serverURL: " https://fleet.example.com " })).toEqual({});
  });

  it("still rejects an invalid URL", () => {
    expect(validate({ serverURL: " fleet example " }).serverURL).toBeDefined();
  });

  // Blur validates the raw value; Save validates the trimmed one. A
  // whitespace-only value counts as empty and must get the same error from both.
  it("treats a whitespace-only URL as empty", () => {
    const data = { serverURL: "   " };

    expect(validate(data).serverURL).toBe("Enter your Fleet web address");
    expect(validate(data)).toEqual(validate(trimFormData(data)));
  });
});
