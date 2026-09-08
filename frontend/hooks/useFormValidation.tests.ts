import { act, renderHook } from "@testing-library/react";

import { notify } from "components/ToastNotification";

import useFormValidation, { IFormErrors } from "./useFormValidation";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

/**
 * These tests call the hook's functions directly — nothing here dispatches a DOM
 * event. How each maps onto the event whose behavior patterns.md specifies:
 *
 * | Event                                        | Function          |
 * | -------------------------------------------- | ----------------- |
 * | a field's `onFocus`                          | `clearFieldError` |
 * | a field's `onBlur`                           | `validateField`   |
 * | a text input's `onChange`                    | `setField`        |
 * | a checkbox/radio/dropdown's `onChange`       | `commitFields`    |
 * | the `<form>`'s `onSubmit`                    | `handleSubmit`    |
 */
type TestFormData = {
  name: string;
  email: string;
  password: string;
  ssoEnabled: boolean;
};

const INITIAL_FORM_DATA: TestFormData = {
  name: "",
  email: "",
  password: "",
  ssoEnabled: false,
};

const validate = (data: TestFormData): IFormErrors => {
  const errors: IFormErrors = {};
  if (!data.name.trim()) {
    errors.name = "Enter a name";
  }
  if (!data.email.trim()) {
    errors.email = "Enter an email";
  } else if (!data.email.includes("@")) {
    errors.email = "Enter a valid email";
  }
  // Conditionally required: a password is only needed when SSO is off.
  if (!data.ssoEnabled && !data.password) {
    errors.password = "Enter a password";
  }
  return errors;
};

/** Props a test can change between renders, the way a parent component would. */
interface IHookProps {
  serverErrors?: IFormErrors;
  isSubmitting?: boolean;
}

const setup = (
  options: { skipTrim?: readonly (keyof TestFormData & string)[] } = {}
) =>
  renderHook(
    ({ serverErrors, isSubmitting }: IHookProps) =>
      useFormValidation<TestFormData>({
        initialFormData: INITIAL_FORM_DATA,
        validate,
        serverErrors,
        isSubmitting,
        skipTrim: options.skipTrim,
      }),
    { initialProps: {} as IHookProps }
  );

type SetupResult = ReturnType<typeof setup>["result"];

const submitEvent = () =>
  (({ preventDefault: jest.fn() } as unknown) as React.FormEvent);

describe("useFormValidation", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("starts with no errors, not dirty and not submitting", () => {
    const { result } = setup();

    expect(result.current.errors).toEqual({});
    expect(result.current.isDirty).toBe(false);
    expect(result.current.isSubmitting).toBe(false);
  });

  // Callers may put any of these in an effect dependency list, so an unstable
  // identity would silently re-run their effect on every render — and loop if
  // that effect sets state. Comments can't enforce that; this can.
  it("keeps every returned callback identity stable across renders", () => {
    const { result, rerender } = setup();
    const first = result.current;

    act(() => result.current.setField("name", "Alice"));
    rerender({ serverErrors: { email: "Enter a different email" } });
    rerender({ isSubmitting: true });

    expect(result.current.setField).toBe(first.setField);
    expect(result.current.commitFields).toBe(first.commitFields);
    expect(result.current.reset).toBe(first.reset);
    expect(result.current.clearFieldError).toBe(first.clearFieldError);
    expect(result.current.validateField).toBe(first.validateField);
    expect(result.current.clearErrors).toBe(first.clearErrors);
    expect(result.current.handleSubmit).toBe(first.handleSubmit);
  });

  it("still submits the latest form data through a stable handleSubmit", () => {
    const onValid = jest.fn();
    const { result } = setup();
    const handleSubmitAtMount = result.current.handleSubmit;

    act(() => result.current.setField("name", "Alice"));
    act(() => result.current.setField("email", "alice@example.com"));
    act(() => result.current.setField("password", "pa55word!"));

    // The mount-time reference must see the values typed after it was created.
    act(() => handleSubmitAtMount(onValid)(submitEvent()));

    expect(onValid).toHaveBeenCalledWith({
      name: "Alice",
      email: "alice@example.com",
      password: "pa55word!",
      ssoEnabled: false,
    });
  });

  describe("when errors appear", () => {
    it("validateField does nothing while the field is pristine", () => {
      const { result } = setup();

      act(() => result.current.validateField("email"));

      expect(result.current.getError("email")).toBeUndefined();
    });

    it("validateField shows the error once the field is dirty", () => {
      const { result } = setup();

      act(() => result.current.setField("email", "nope"));
      act(() => result.current.validateField("email"));

      expect(result.current.getError("email")).toBe("Enter a valid email");
    });

    it("setField keeps the field dirty after its value returns to the initial one", () => {
      const { result } = setup();

      act(() => result.current.setField("email", "a"));
      act(() => result.current.setField("email", ""));
      act(() => result.current.validateField("email"));

      expect(result.current.getError("email")).toBe("Enter an email");
    });

    it("validateField touches only its own field", () => {
      const { result } = setup();

      act(() => result.current.setField("email", "nope"));
      act(() => result.current.validateField("email"));

      expect(result.current.errors).toEqual({ email: "Enter a valid email" });
    });

    it("handleSubmit reveals every error, regardless of dirty state", () => {
      const onValid = jest.fn();
      const { result } = setup();

      act(() => result.current.handleSubmit(onValid)(submitEvent()));

      expect(result.current.errors).toEqual({
        name: "Enter a name",
        email: "Enter an email",
        password: "Enter a password",
      });
      expect(onValid).not.toHaveBeenCalled();
    });

    it("validateField works on a field that only handleSubmit marked dirty", () => {
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      act(() => result.current.clearFieldError("name"));
      expect(result.current.getError("name")).toBeUndefined();

      act(() => result.current.validateField("name"));

      expect(result.current.getError("name")).toBe("Enter a name");
    });
  });

  describe("when errors clear", () => {
    it("clearFieldError drops the error without waiting for a valid value", () => {
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      act(() => result.current.clearFieldError("email"));

      expect(result.current.getError("email")).toBeUndefined();
      expect(result.current.getError("name")).toBe("Enter a name");
    });

    it("setField does not re-validate", () => {
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      // A valid value alone must not clear the error — that takes an explicit
      // clearFieldError or validateField.
      act(() => result.current.setField("email", "user@example.com"));

      expect(result.current.getError("email")).toBe("Enter an email");
    });

    it("setField never clears another field's error", () => {
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      act(() => result.current.setField("name", "Alice"));

      expect(result.current.getError("email")).toBe("Enter an email");
      expect(result.current.getError("password")).toBe("Enter a password");
    });

    it("commitFields drops an error its change made irrelevant, and leaves the rest", () => {
      const { result } = setup();

      // Reveal all three errors first, so there is something to drop.
      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      expect(result.current.getError("password")).toBe("Enter a password");

      act(() => result.current.commitFields({ ssoEnabled: true }));

      expect(result.current.getError("password")).toBeUndefined();
      expect(result.current.getError("name")).toBe("Enter a name");
    });

    it("commitFields adds no error on a form whose errors have not been revealed", () => {
      const { result } = setup();

      // Nothing has been revealed: name and email are invalid, but their errors
      // are not shown, and committing a change must not surface them.
      act(() => result.current.commitFields({ ssoEnabled: true }));

      expect(result.current.errors).toEqual({});
    });

    it("commitFields keeps the message already on screen when the field is still invalid", () => {
      const { result, rerender } = setup();

      rerender({ serverErrors: { email: "Enter an email nobody else uses" } });
      act(() => result.current.commitFields({ ssoEnabled: true }));

      expect(result.current.getError("email")).toBe(
        "Enter an email nobody else uses"
      );
    });

    it("clearErrors and reset drop every error", () => {
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      act(() => result.current.clearErrors());
      expect(result.current.errors).toEqual({});

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      act(() =>
        result.current.reset({ ...INITIAL_FORM_DATA, name: "Reset name" })
      );

      expect(result.current.errors).toEqual({});
      expect(result.current.formData.name).toBe("Reset name");
      expect(result.current.isDirty).toBe(false);
    });
  });

  describe("handleSubmit", () => {
    const validData = {
      name: "  Alice  ",
      email: "  alice@example.com  ",
      password: "  pa55word!  ",
    };

    const fillIn = (result: SetupResult) => {
      act(() => result.current.setField("name", validData.name));
      act(() => result.current.setField("email", validData.email));
      act(() => result.current.setField("password", validData.password));
    };

    it("calls the callback with trimmed values", () => {
      const onValid = jest.fn();
      const { result } = setup();
      fillIn(result);

      act(() => result.current.handleSubmit(onValid)(submitEvent()));

      expect(onValid).toHaveBeenCalledWith({
        name: "Alice",
        email: "alice@example.com",
        password: "pa55word!",
        ssoEnabled: false,
      });
    });

    it("leaves skipTrim fields verbatim", () => {
      const onValid = jest.fn();
      const { result } = setup({ skipTrim: ["password"] });
      fillIn(result);

      act(() => result.current.handleSubmit(onValid)(submitEvent()));

      expect(onValid).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Alice", password: "  pa55word!  " })
      );
    });

    it("calls preventDefault", () => {
      const evt = submitEvent();
      const { result } = setup();

      act(() => result.current.handleSubmit(jest.fn())(evt));

      expect(evt.preventDefault).toHaveBeenCalled();
    });

    it("tracks in-flight state for a promise-returning callback", async () => {
      let resolveSubmit: () => void = () => undefined;
      const onValid = jest.fn(
        () =>
          new Promise<void>((resolve) => {
            resolveSubmit = resolve;
          })
      );
      const { result } = setup();
      fillIn(result);

      act(() => result.current.handleSubmit(onValid)(submitEvent()));
      expect(result.current.isSubmitting).toBe(true);

      await act(async () => {
        resolveSubmit();
      });

      expect(result.current.isSubmitting).toBe(false);
    });

    it("ends in-flight state when the callback rejects", async () => {
      const onValid = jest.fn(() => Promise.reject(new Error("nope")));
      const { result } = setup();
      fillIn(result);

      await act(async () => {
        result.current.handleSubmit(onValid)(submitEvent());
      });

      expect(result.current.isSubmitting).toBe(false);
    });

    it("drops a second submit while one is in flight", () => {
      const onValid = jest.fn(() => new Promise<void>(() => undefined));
      const { result } = setup();
      fillIn(result);

      act(() => result.current.handleSubmit(onValid)(submitEvent()));
      act(() => result.current.handleSubmit(onValid)(submitEvent()));

      expect(onValid).toHaveBeenCalledTimes(1);
    });

    it("drops a submit while a parent-owned request is in flight", () => {
      const onValid = jest.fn();
      const { result, rerender } = setup();
      fillIn(result);

      rerender({ isSubmitting: true });
      expect(result.current.isSubmitting).toBe(true);

      act(() => result.current.handleSubmit(onValid)(submitEvent()));

      expect(onValid).not.toHaveBeenCalled();
    });
  });

  describe("serverErrors", () => {
    it("exposes each error through getError and fires one toast each", () => {
      const { result, rerender } = setup();

      rerender({
        serverErrors: {
          email: "Enter an email that isn't already in use",
          password: "Enter a password that meets the requirements below",
        },
      });

      expect(result.current.getError("email")).toBe(
        "Enter an email that isn't already in use"
      );
      expect(result.current.getError("password")).toBe(
        "Enter a password that meets the requirements below"
      );
      expect(notify.error).toHaveBeenCalledTimes(2);
    });

    it("survive a commitFields that the client rule cannot reproduce", () => {
      const { result, rerender } = setup();

      // A well-formed but already-taken email: nothing for `validate` to flag,
      // so pruning would silently drop the server's verdict.
      act(() => result.current.setField("email", "taken@example.com"));
      rerender({
        serverErrors: { email: "Enter an email that isn't already in use" },
      });

      act(() => result.current.commitFields({ ssoEnabled: true }));

      expect(result.current.getError("email")).toBe(
        "Enter an email that isn't already in use"
      );
    });

    it("are replaced wholesale by the client verdict on submit", () => {
      const { result, rerender } = setup();

      act(() => result.current.setField("email", "taken@example.com"));
      rerender({
        serverErrors: { email: "Enter an email that isn't already in use" },
      });

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));

      // Name and password are still empty, so submit fails — but on its own
      // verdict, not the stale server one.
      expect(result.current.getError("email")).toBeUndefined();
      expect(result.current.getError("name")).toBe("Enter a name");
    });

    it("clearFieldError drops a server error, like a client-side one", () => {
      const { result, rerender } = setup();

      rerender({ serverErrors: { email: "Enter a different email" } });
      act(() => result.current.clearFieldError("email"));

      expect(result.current.getError("email")).toBeUndefined();
    });

    it("are applied once even if the same content arrives on every render", () => {
      const { result, rerender } = setup();

      // This effect sets state, so re-reacting to unchanged content would loop.
      rerender({ serverErrors: { email: "Enter a different email" } });
      rerender({ serverErrors: { email: "Enter a different email" } });
      rerender({ serverErrors: { email: "Enter a different email" } });

      expect(result.current.getError("email")).toBe("Enter a different email");
      expect(notify.error).toHaveBeenCalledTimes(1);
    });

    it("toast again on a genuinely repeated failure", () => {
      const { result, rerender } = setup();
      const failure = { email: "Enter a different email" };

      act(() => result.current.setField("name", "Alice"));
      act(() => result.current.setField("email", "taken@example.com"));
      act(() => result.current.setField("password", "pa55word!"));

      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      rerender({ serverErrors: { ...failure } });
      expect(notify.error).toHaveBeenCalledTimes(1);

      // Re-submitting clears the shown errors, so the same failure is new again.
      act(() => result.current.handleSubmit(jest.fn())(submitEvent()));
      rerender({ serverErrors: { ...failure } });

      expect(notify.error).toHaveBeenCalledTimes(2);
      expect(result.current.getError("email")).toBe("Enter a different email");
    });

    it("does not toast an empty batch", () => {
      const { result, rerender } = setup();

      rerender({ serverErrors: {} });

      expect(result.current.errors).toEqual({});
      expect(notify.error).not.toHaveBeenCalled();
    });
  });

  describe("form data", () => {
    it("tracks changes through setField and commitFields", () => {
      const { result } = setup();

      act(() => result.current.setField("name", "Alice"));
      act(() => result.current.commitFields({ ssoEnabled: true }));

      expect(result.current.formData).toEqual({
        ...INITIAL_FORM_DATA,
        name: "Alice",
        ssoEnabled: true,
      });
      expect(result.current.isDirty).toBe(true);
    });
  });
});
