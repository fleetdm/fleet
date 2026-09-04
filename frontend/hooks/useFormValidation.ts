import React, { useCallback, useEffect, useRef, useState } from "react";

import { notify } from "components/ToastNotification";

/**
 * `fieldName` -> error message. Only invalid fields appear.
 *
 * Keys are free-form strings rather than `keyof TFormData` because some errors
 * belong to a selector rather than a single form value (e.g. `teams` for an
 * "at least one fleet" rule).
 */
export type IFormErrors = Record<string, string>;

/**
 * Pure function: current form data in, `fieldName` -> message out. Must not
 * read component state other than what it closes over, and must not fire side
 * effects (no toasts) — the hook calls it on every blur and submit.
 *
 * Does not need to be memoized; the hook always calls the latest one.
 */
export type ValidateFn<TFormData> = (formData: TFormData) => IFormErrors;

interface IUseFormValidationOptions<TFormData> {
  initialFormData: TFormData;
  validate: ValidateFn<TFormData>;
  /**
   * Field-specific errors returned by the API, keyed the way `validate` keys
   * its output. Pass a new object per failed request: each batch is shown
   * inline and toasted once.
   */
  serverErrors?: IFormErrors | null;
  /**
   * In-flight state owned by a parent (a page that holds the request promise
   * and passes `isSubmitting` down). OR-ed with the hook's own in-flight state
   * for the returned `isSubmitting` and the double-submit guard.
   */
  isSubmitting?: boolean;
  /**
   * String fields that must reach the API verbatim. Everything else is trimmed
   * on submit. Passwords belong here — trimming changes the credential.
   */
  skipTrim?: readonly (keyof TFormData & string)[];
}

interface IUseFormValidationReturn<TFormData> {
  formData: TFormData;
  /**
   * For text inputs (`onChange`). Marks the field dirty and deliberately leaves
   * errors alone: errors surface on blur/submit and clear on focus, never on
   * keystroke. Typing in one field also never clears another field's error.
   */
  setField: <K extends keyof TFormData & string>(
    name: K,
    value: TFormData[K]
  ) => void;
  /**
   * For controls whose interaction is complete on change (checkbox, radio,
   * dropdown, multi-select) and for programmatic updates. Applies the change
   * and immediately drops any shown error the change made irrelevant — e.g.
   * toggling SSO on clears the password error. Never adds new errors.
   */
  commitFields: (changes: Partial<TFormData>) => void;
  /** Replaces form data and clears all errors and dirty state. */
  reset: (data: TFormData) => void;
  /** Currently shown errors. For a single field prefer `getError`. */
  errors: IFormErrors;
  /** Feeds a field's `error` prop. */
  getError: (name: string) => string | undefined;
  /** Wire to a field's `onFocus`. Clears that field's error, client or server. */
  clearFieldError: (name: string) => void;
  /**
   * Wire to a field's `onBlur`. Validates that one field, and only once it is
   * dirty, so a pristine required field stays silent until submit.
   */
  validateField: (name: string) => void;
  clearErrors: () => void;
  /**
   * Wire to the `<form>`'s `onSubmit` — not to the button's `onClick`. Calls
   * `preventDefault`, drops a submit that arrives while one is in flight,
   * validates every field, and on any error shows them all and returns without
   * invoking `onValid`. `onValid` receives trimmed data (see `skipTrim`);
   * return its promise to have the hook track in-flight state.
   */
  handleSubmit: (
    onValid: (formData: TFormData) => void | Promise<unknown>
  ) => (evt: React.FormEvent) => void;
  /** Disable the submit button and the fields on this, and nothing else. */
  isSubmitting: boolean;
  /**
   * "Form has changes." Never gate the submit button on this — a no-op re-save
   * is allowed. Useful for unsaved-changes navigation guards.
   */
  isDirty: boolean;
}

const trimFormData = <TFormData>(
  formData: TFormData,
  skipTrim: readonly string[]
): TFormData => {
  const trimmed = { ...formData } as Record<string, unknown>;
  Object.keys(trimmed).forEach((key) => {
    const value = trimmed[key];
    if (typeof value === "string" && !skipTrim.includes(key)) {
      trimmed[key] = value.trim();
    }
  });
  return trimmed as TFormData;
};

const NO_SKIP_TRIM: readonly never[] = [];

/**
 * Single source of truth for the form validation behavior specified in
 * `frontend/docs/patterns.md#data-validation`. Read that first — the rules it
 * documents (errors clear on focus, blur validates one dirty field, submit is a
 * checkpoint that reveals everything, the submit button stays enabled) are the
 * reason this hook's setters are split the way they are.
 *
 * Notably absent: an `isValid` flag. The submit button must not be disabled for
 * invalid values, so nothing should need one.
 *
 * ```tsx
 * const validate = (data: IFormData): IFormErrors => {
 *   const errors: IFormErrors = {};
 *   if (!validatePresence(data.name)) errors.name = "Enter a name";
 *   return errors;
 * };
 *
 * const {
 *   formData, setField, getError, clearFieldError, validateField, handleSubmit,
 *   isSubmitting,
 * } = useFormValidation({ initialFormData: { name: "" }, validate });
 *
 * return (
 *   <form onSubmit={handleSubmit(onSave)}>
 *     <InputField
 *       name="name"
 *       value={formData.name}
 *       error={getError("name")}
 *       onChange={(value: string) => setField("name", value)}
 *       onFocus={() => clearFieldError("name")}
 *       onBlur={() => validateField("name")}
 *       disabled={isSubmitting}
 *     />
 *     <Button type="submit" isLoading={isSubmitting} disabled={isSubmitting}>
 *       Save
 *     </Button>
 *   </form>
 * );
 * ```
 */
const useFormValidation = <TFormData extends object>({
  initialFormData,
  validate,
  serverErrors,
  isSubmitting: isSubmittingExternal = false,
  skipTrim = NO_SKIP_TRIM,
}: IUseFormValidationOptions<TFormData>): IUseFormValidationReturn<TFormData> => {
  const [formData, setFormData] = useState<TFormData>(initialFormData);
  const [errors, setErrors] = useState<IFormErrors>({});
  const [isSubmittingInternal, setIsSubmittingInternal] = useState(false);
  const [isDirty, setIsDirty] = useState(false);

  const formDataRef = useRef(formData);
  const dirtyFieldsRef = useRef(new Set<string>());
  // Fields whose shown error came from the API. Client validation can't
  // reproduce those (a duplicate email is still a well-formed email), so they
  // are exempt from pruning and only leave on focus, submit, reset or clear.
  const serverErrorFieldsRef = useRef(new Set<string>());
  const isSubmittingRef = useRef(false);

  const validateRef = useRef(validate);
  const skipTrimRef = useRef(skipTrim);
  const isSubmittingExternalRef = useRef(isSubmittingExternal);
  const errorsRef = useRef(errors);

  // Synced after commit rather than during render: a render React discards must
  // not leave a ref pointing at values that never shipped.
  //
  // No dependency array, so this runs on every commit — which is safe only
  // because the body does nothing but assign refs. Ref writes don't schedule a
  // render, so there is nothing to loop. Never add a state update here.
  useEffect(() => {
    validateRef.current = validate;
    skipTrimRef.current = skipTrim;
    isSubmittingExternalRef.current = isSubmittingExternal;
    errorsRef.current = errors;
  });

  // Field-specific server errors render inline AND toast: a long form can
  // scroll the errored field off-screen by the time the response lands.
  useEffect(() => {
    if (!serverErrors) {
      return;
    }
    const incoming: IFormErrors = {};
    Object.keys(serverErrors).forEach((key) => {
      const message = serverErrors[key];
      if (message) {
        incoming[key] = message;
      }
    });
    // Surface only what isn't already on screen. This effect sets state, so
    // reacting to a prop whose identity changes without its content changing
    // would loop — which is exactly what a caller passing an inline object
    // literal does. A genuinely repeated failure still toasts again, because
    // handleSubmit clears the shown errors before each attempt.
    const fresh = Object.keys(incoming).filter(
      (field) => errorsRef.current[field] !== incoming[field]
    );
    if (!fresh.length) {
      return;
    }
    fresh.forEach((field) => serverErrorFieldsRef.current.add(field));
    setErrors((prev) => {
      const next = { ...prev };
      fresh.forEach((field) => {
        next[field] = incoming[field];
      });
      return next;
    });
    fresh.forEach((field) => notify.error(incoming[field]));
  }, [serverErrors]);

  const setField = useCallback(
    <K extends keyof TFormData & string>(name: K, value: TFormData[K]) => {
      const next = ({
        ...formDataRef.current,
        [name]: value,
      } as unknown) as TFormData;
      formDataRef.current = next;
      setFormData(next);
      dirtyFieldsRef.current.add(name);
      setIsDirty(true);
    },
    []
  );

  const commitFields = useCallback((changes: Partial<TFormData>) => {
    const next = { ...formDataRef.current, ...changes };
    formDataRef.current = next;
    setFormData(next);
    Object.keys(changes).forEach((key) => dirtyFieldsRef.current.add(key));
    setIsDirty(true);

    const currentErrors = validateRef.current(next);
    setErrors((prev) => {
      const kept: IFormErrors = {};
      let dropped = false;
      Object.keys(prev).forEach((key) => {
        if (currentErrors[key] || serverErrorFieldsRef.current.has(key)) {
          // Keep the message already on screen rather than the freshly computed
          // one — a server error must not be overwritten by a client rule.
          kept[key] = prev[key];
        } else {
          dropped = true;
        }
      });
      return dropped ? kept : prev;
    });
  }, []);

  const reset = useCallback((data: TFormData) => {
    formDataRef.current = data;
    dirtyFieldsRef.current = new Set<string>();
    serverErrorFieldsRef.current = new Set<string>();
    setFormData(data);
    setErrors({});
    setIsDirty(false);
  }, []);

  const getError = useCallback((name: string) => errors[name], [errors]);

  const clearFieldError = useCallback((name: string) => {
    serverErrorFieldsRef.current.delete(name);
    setErrors((prev) => {
      if (!(name in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[name];
      return next;
    });
  }, []);

  const validateField = useCallback((name: string) => {
    if (!dirtyFieldsRef.current.has(name)) {
      return;
    }
    // Blur hands the field back to client validation, so a server verdict on it
    // no longer applies.
    serverErrorFieldsRef.current.delete(name);
    const message = validateRef.current(formDataRef.current)[name];
    setErrors((prev) => {
      if (prev[name] === message || (!message && !(name in prev))) {
        return prev;
      }
      const next = { ...prev };
      if (message) {
        next[name] = message;
      } else {
        delete next[name];
      }
      return next;
    });
  }, []);

  const clearErrors = useCallback(() => {
    serverErrorFieldsRef.current = new Set<string>();
    setErrors({});
  }, []);

  const handleSubmit = useCallback(
    (onValid: (formData: TFormData) => void | Promise<unknown>) => (
      evt: React.FormEvent
    ) => {
      evt.preventDefault();

      // The disabled button is not the only guard — Enter in a text field and a
      // second click landing before the re-render both reach this handler.
      if (isSubmittingRef.current || isSubmittingExternalRef.current) {
        return;
      }

      const submitData = trimFormData(formDataRef.current, skipTrimRef.current);
      const submitErrors = validateRef.current(submitData);

      // Both branches below replace the whole error map with what `validate`
      // just returned, so a server verdict the client rules can't re-derive is
      // dropped here — a new request is about to supersede it anyway. Forget the
      // exemptions too, or `commitFields` would keep protecting fields that no
      // longer hold a server error.
      serverErrorFieldsRef.current = new Set<string>();

      if (Object.keys(submitErrors).length) {
        // Submit is a checkpoint: it bypasses the dirty gate and reveals every
        // error at once. Marking those fields dirty keeps blur re-validation
        // working on the fields the user has yet to touch.
        Object.keys(submitErrors).forEach((field) =>
          dirtyFieldsRef.current.add(field)
        );
        setErrors(submitErrors);
        return;
      }

      setErrors({});

      const result = onValid(submitData);
      if (!result || typeof result.then !== "function") {
        return;
      }

      isSubmittingRef.current = true;
      setIsSubmittingInternal(true);
      // `onValid` owns reporting its own failure; a rejection here only ends the
      // in-flight state so the fields and the button come back.
      const settle = () => {
        isSubmittingRef.current = false;
        setIsSubmittingInternal(false);
      };
      result.then(settle, settle);
    },
    []
  );

  return {
    formData,
    setField,
    commitFields,
    reset,
    errors,
    getError,
    clearFieldError,
    validateField,
    clearErrors,
    handleSubmit,
    isSubmitting: isSubmittingInternal || isSubmittingExternal,
    isDirty,
  };
};

export default useFormValidation;
