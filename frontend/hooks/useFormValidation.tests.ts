import { renderHook, act } from "@testing-library/react";

import {
  runValidation,
  useFormValidation,
  IValidationConfig,
} from "./useFormValidation";

// --- Test fixtures ---

interface ITestFormData {
  name: string;
  description: string;
}

const REQUIRED_CONFIG: IValidationConfig<ITestFormData> = {
  name: {
    validations: [
      {
        name: "required",
        isValid: (fd) => fd.name.length > 0,
        message: "Name is required",
      },
    ],
  },
  description: {
    validations: [
      {
        name: "required",
        isValid: (fd) => fd.description.length > 0,
        message: "Description is required",
      },
    ],
  },
};

// --- runValidation tests ---

describe("runValidation", () => {
  it("returns isValid true when all fields pass", () => {
    const result = runValidation(
      { name: "hello", description: "world" },
      REQUIRED_CONFIG
    );
    expect(result.isValid).toBe(true);
    expect(result.fields.name).toEqual({ isValid: true });
    expect(result.fields.description).toEqual({ isValid: true });
  });

  it("returns isValid false with correct message for failing fields", () => {
    const result = runValidation(
      { name: "", description: "world" },
      REQUIRED_CONFIG
    );
    expect(result.isValid).toBe(false);
    expect(result.fields.name).toEqual({
      isValid: false,
      message: "Name is required",
    });
    expect(result.fields.description).toEqual({ isValid: true });
  });

  it("reports multiple failing fields", () => {
    const result = runValidation(
      { name: "", description: "" },
      REQUIRED_CONFIG
    );
    expect(result.isValid).toBe(false);
    expect(result.fields.name?.isValid).toBe(false);
    expect(result.fields.description?.isValid).toBe(false);
  });

  it("uses first failing rule (rule precedence)", () => {
    const config: IValidationConfig<ITestFormData> = {
      name: {
        validations: [
          {
            name: "required",
            isValid: (fd) => fd.name.length > 0,
            message: "Name is required",
          },
          {
            name: "maxLength",
            isValid: (fd) => fd.name.length <= 10,
            message: "Name too long",
          },
        ],
      },
    };

    // Empty triggers "required" (first rule), not "maxLength"
    const resultEmpty = runValidation({ name: "", description: "" }, config);
    expect(resultEmpty.fields.name?.message).toBe("Name is required");

    // Long name triggers "maxLength" (second rule)
    const resultLong = runValidation(
      { name: "a".repeat(11), description: "" },
      config
    );
    expect(resultLong.fields.name?.message).toBe("Name too long");
  });

  it("resolves dynamic messages (function-based)", () => {
    const config: IValidationConfig<ITestFormData> = {
      name: {
        validations: [
          {
            name: "maxLength",
            isValid: (fd) => fd.name.length <= 5,
            message: (fd) => `Name "${fd.name}" exceeds 5 characters`,
          },
        ],
      },
    };

    const result = runValidation({ name: "toolong", description: "" }, config);
    expect(result.fields.name?.message).toBe(
      'Name "toolong" exceeds 5 characters'
    );
  });

  it("supports cross-field validation via currentValidation parameter", () => {
    interface ITimeForm {
      startTime: string;
      endTime: string;
    }

    const config: IValidationConfig<ITimeForm> = {
      startTime: {
        validations: [
          {
            name: "required",
            isValid: (fd) => fd.startTime.length > 0,
            message: "Start time required",
          },
        ],
      },
      endTime: {
        validations: [
          {
            name: "afterStart",
            isValid: (fd, currentValidation) => {
              // Skip if startTime failed
              if (
                currentValidation?.startTime &&
                !currentValidation.startTime.isValid
              ) {
                return true;
              }
              return fd.endTime > fd.startTime;
            },
            message: "End must be after start",
          },
        ],
      },
    };

    // startTime invalid → endTime validation skipped (returns true)
    const resultSkipped = runValidation(
      { startTime: "", endTime: "01:00" },
      config
    );
    expect(resultSkipped.fields.endTime?.isValid).toBe(true);

    // Both valid, endTime before startTime → fails
    const resultFailed = runValidation(
      { startTime: "10:00", endTime: "09:00" },
      config
    );
    expect(resultFailed.fields.endTime).toEqual({
      isValid: false,
      message: "End must be after start",
    });

    // Both valid, endTime after startTime → passes
    const resultPassed = runValidation(
      { startTime: "09:00", endTime: "10:00" },
      config
    );
    expect(resultPassed.isValid).toBe(true);
  });

  it("skips fields when shouldValidateField returns false", () => {
    const result = runValidation(
      { name: "", description: "" },
      REQUIRED_CONFIG,
      (field) => field !== "description"
    );
    expect(result.fields.name?.isValid).toBe(false);
    expect(result.fields.description).toBeUndefined();
    expect(result.isValid).toBe(false);
  });

  it("passes formData to shouldValidateField", () => {
    const result = runValidation(
      { name: "", description: "" },
      REQUIRED_CONFIG,
      (_field, fd) => fd.name.length > 0 // skip all when name is empty
    );
    expect(result.isValid).toBe(true);
    expect(result.fields.name).toBeUndefined();
    expect(result.fields.description).toBeUndefined();
  });

  it("handles validation rule with no message", () => {
    const config: IValidationConfig<ITestFormData> = {
      name: {
        validations: [
          {
            name: "required",
            isValid: (fd) => fd.name.length > 0,
            // no message
          },
        ],
      },
    };

    const result = runValidation({ name: "", description: "" }, config);
    expect(result.fields.name).toEqual({ isValid: false, message: undefined });
  });

  it("returns isValid true for empty config", () => {
    const result = runValidation({ name: "", description: "" }, {});
    expect(result.isValid).toBe(true);
  });
});

// --- useFormValidation hook tests ---

describe("useFormValidation", () => {
  const initialFormData: ITestFormData = { name: "", description: "" };

  it("initializes with correct formData and no shown errors", () => {
    const { result } = renderHook(() =>
      useFormValidation({
        initialFormData,
        validationConfig: REQUIRED_CONFIG,
      })
    );

    expect(result.current.formData).toEqual(initialFormData);
    expect(result.current.isValid).toBe(true);
    expect(result.current.getFieldError("name")).toBeUndefined();
    expect(result.current.getFieldError("description")).toBeUndefined();
  });

  describe("setField", () => {
    it("updates formData for the given field", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => result.current.setField("name", "hello"));

      expect(result.current.formData.name).toBe("hello");
      expect(result.current.formData.description).toBe("");
    });

    it("does NOT add new errors", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      // Set an invalid value — no error should appear
      act(() => result.current.setField("name", ""));

      expect(result.current.getFieldError("name")).toBeUndefined();
    });

    it("clears an existing error when the field becomes valid", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      // Show all errors first via validateAll
      act(() => result.current.validateAll());
      expect(result.current.getFieldError("name")).toBe("Name is required");

      // Fix the name field
      act(() => result.current.setField("name", "hello"));

      expect(result.current.getFieldError("name")).toBeUndefined();
    });

    it("does NOT clear error if field is still invalid", () => {
      const config: IValidationConfig<ITestFormData> = {
        name: {
          validations: [
            {
              name: "minLength",
              isValid: (fd) => fd.name.length >= 3,
              message: "Name must be at least 3 characters",
            },
          ],
        },
      };

      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: config,
        })
      );

      // Show errors
      act(() => result.current.validateAll());
      expect(result.current.getFieldError("name")).toBe(
        "Name must be at least 3 characters"
      );

      // Still invalid (only 2 chars)
      act(() => result.current.setField("name", "ab"));
      expect(result.current.getFieldError("name")).toBe(
        "Name must be at least 3 characters"
      );

      // Now valid
      act(() => result.current.setField("name", "abc"));
      expect(result.current.getFieldError("name")).toBeUndefined();
    });

    it("only clears the changed field's error, not others", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      // Show all errors
      act(() => result.current.validateAll());
      expect(result.current.getFieldError("name")).toBe("Name is required");
      expect(result.current.getFieldError("description")).toBe(
        "Description is required"
      );

      // Fix name only
      act(() => result.current.setField("name", "hello"));

      expect(result.current.getFieldError("name")).toBeUndefined();
      expect(result.current.getFieldError("description")).toBe(
        "Description is required"
      );
    });
  });

  describe("validateAll", () => {
    it("shows all errors for invalid fields", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => result.current.validateAll());

      expect(result.current.isValid).toBe(false);
      expect(result.current.getFieldError("name")).toBe("Name is required");
      expect(result.current.getFieldError("description")).toBe(
        "Description is required"
      );
    });

    it("shows no errors when all fields are valid", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData: { name: "hello", description: "world" },
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => result.current.validateAll());

      expect(result.current.isValid).toBe(true);
      expect(result.current.getFieldError("name")).toBeUndefined();
      expect(result.current.getFieldError("description")).toBeUndefined();
    });
  });

  describe("handleSubmit", () => {
    it("calls preventDefault on the event", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData: { name: "hello", description: "world" },
          validationConfig: REQUIRED_CONFIG,
        })
      );

      const callback = jest.fn();
      const preventDefault = jest.fn();
      const handler = result.current.handleSubmit(callback);

      act(() =>
        handler(({
          preventDefault,
        } as unknown) as React.FormEvent<HTMLFormElement>)
      );

      expect(preventDefault).toHaveBeenCalled();
    });

    it("calls callback with formData when valid", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData: { name: "hello", description: "world" },
          validationConfig: REQUIRED_CONFIG,
        })
      );

      const callback = jest.fn();
      const handler = result.current.handleSubmit(callback);

      act(() =>
        handler(({
          preventDefault: jest.fn(),
        } as unknown) as React.FormEvent<HTMLFormElement>)
      );

      expect(callback).toHaveBeenCalledWith({
        name: "hello",
        description: "world",
      });
    });

    it("does NOT call callback when invalid, and shows all errors", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      const callback = jest.fn();
      const handler = result.current.handleSubmit(callback);

      act(() =>
        handler(({
          preventDefault: jest.fn(),
        } as unknown) as React.FormEvent<HTMLFormElement>)
      );

      expect(callback).not.toHaveBeenCalled();
      expect(result.current.getFieldError("name")).toBe("Name is required");
      expect(result.current.getFieldError("description")).toBe(
        "Description is required"
      );
    });

    it("uses latest formData after setField calls", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => {
        result.current.setField("name", "hello");
        result.current.setField("description", "world");
      });

      const callback = jest.fn();
      const handler = result.current.handleSubmit(callback);

      act(() =>
        handler(({
          preventDefault: jest.fn(),
        } as unknown) as React.FormEvent<HTMLFormElement>)
      );

      expect(callback).toHaveBeenCalledWith({
        name: "hello",
        description: "world",
      });
    });
  });

  describe("setFormData", () => {
    it("shows all current errors after update", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData: { name: "hello", description: "world" },
          validationConfig: REQUIRED_CONFIG,
        })
      );

      // Replace with invalid data
      act(() => result.current.setFormData({ name: "", description: "world" }));

      expect(result.current.getFieldError("name")).toBe("Name is required");
      expect(result.current.formData).toEqual({
        name: "",
        description: "world",
      });
    });

    it("accepts a functional updater", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData: { name: "hello", description: "world" },
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() =>
        result.current.setFormData((prev) => ({ ...prev, name: "updated" }))
      );

      expect(result.current.formData.name).toBe("updated");
    });
  });

  describe("clearErrors", () => {
    it("resets all shown errors", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      // Show errors
      act(() => result.current.validateAll());
      expect(result.current.isValid).toBe(false);

      // Clear them
      act(() => result.current.clearErrors());

      expect(result.current.isValid).toBe(true);
      expect(result.current.getFieldError("name")).toBeUndefined();
      expect(result.current.getFieldError("description")).toBeUndefined();
    });
  });

  describe("isValid", () => {
    it("starts true when no errors are shown", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      expect(result.current.isValid).toBe(true);
    });

    it("becomes false after showing errors for invalid form", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => result.current.validateAll());
      expect(result.current.isValid).toBe(false);
    });

    it("becomes true again after fixing all errors", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
        })
      );

      act(() => result.current.validateAll());
      expect(result.current.isValid).toBe(false);

      act(() => result.current.setField("name", "hello"));
      act(() => result.current.setField("description", "world"));

      expect(result.current.isValid).toBe(true);
    });
  });

  describe("validationConfig changes", () => {
    it("clears resolved errors when config changes", () => {
      const strictConfig: IValidationConfig<ITestFormData> = {
        name: {
          validations: [
            {
              name: "minLength",
              isValid: (fd) => fd.name.length >= 5,
              message: "Name must be at least 5 characters",
            },
          ],
        },
      };

      const lenientConfig: IValidationConfig<ITestFormData> = {
        name: {
          validations: [
            {
              name: "minLength",
              isValid: (fd) => fd.name.length >= 1,
              message: "Name is required",
            },
          ],
        },
      };

      const { result, rerender } = renderHook(
        ({ config }) =>
          useFormValidation({
            initialFormData: { name: "abc", description: "" },
            validationConfig: config,
          }),
        { initialProps: { config: strictConfig } }
      );

      // Show error under strict config
      act(() => result.current.validateAll());
      expect(result.current.getFieldError("name")).toBe(
        "Name must be at least 5 characters"
      );

      // Switch to lenient config - "abc" now passes
      rerender({ config: lenientConfig });

      expect(result.current.getFieldError("name")).toBeUndefined();
    });

    it("does NOT add new errors when config changes", () => {
      const lenientConfig: IValidationConfig<ITestFormData> = {
        name: {
          validations: [
            {
              name: "minLength",
              isValid: (fd) => fd.name.length >= 1,
              message: "Name is required",
            },
          ],
        },
      };

      const strictConfig: IValidationConfig<ITestFormData> = {
        name: {
          validations: [
            {
              name: "minLength",
              isValid: (fd) => fd.name.length >= 5,
              message: "Name must be at least 5 characters",
            },
          ],
        },
      };

      const { result, rerender } = renderHook(
        ({ config }) =>
          useFormValidation({
            initialFormData: { name: "abc", description: "" },
            validationConfig: config,
          }),
        { initialProps: { config: lenientConfig } }
      );

      // No errors shown
      expect(result.current.getFieldError("name")).toBeUndefined();

      // Switch to strict config - "abc" now fails, but error should not appear
      rerender({ config: strictConfig });

      expect(result.current.getFieldError("name")).toBeUndefined();
    });
  });

  describe("shouldValidateField", () => {
    it("skips fields where shouldValidateField returns false", () => {
      const { result } = renderHook(() =>
        useFormValidation({
          initialFormData,
          validationConfig: REQUIRED_CONFIG,
          shouldValidateField: (field) => field !== "description",
        })
      );

      act(() => result.current.validateAll());

      expect(result.current.getFieldError("name")).toBe("Name is required");
      expect(result.current.getFieldError("description")).toBeUndefined();
    });
  });
});
