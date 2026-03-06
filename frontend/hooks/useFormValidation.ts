import { useState, useCallback, useRef, useEffect } from "react";

// --- Shared types (exported for consumption by form helpers.ts files) ---

export interface IFieldValidation {
  isValid: boolean;
  message?: string;
}

export type IValidationMessage<TFormData> =
  | string
  | ((formData: TFormData) => string);

export interface IValidationRule<TFormData> {
  name: string;
  isValid: (
    formData: TFormData,
    currentValidation: Record<string, IFieldValidation | undefined>
  ) => boolean;
  message?: IValidationMessage<TFormData>;
}

export type IValidationConfig<TFormData> = Record<
  string,
  { validations: IValidationRule<TFormData>[] }
>;

export interface IValidationResult {
  isValid: boolean;
  fields: Record<string, IFieldValidation | undefined>;
}

// --- Hook options and return types ---

interface IUseFormValidationOptions<TFormData> {
  initialFormData: TFormData;
  validationConfig: IValidationConfig<TFormData>;
  /** Called before validating each field. Return false to skip that field entirely. */
  shouldValidateField?: (fieldKey: string, formData: TFormData) => boolean;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
interface IUseFormValidationReturn<TFormData extends Record<string, any>> {
  formData: TFormData;
  /** Escape hatch: replaces formData and shows all current errors. */
  setFormData: (
    dataOrUpdater: TFormData | ((prev: TFormData) => TFormData)
  ) => void;
  /** True when no shown errors exist. */
  isValid: boolean;
  /** Returns the error message for a field, or undefined if no error is shown. */
  getFieldError: (field: string) => string | undefined;
  /** Updates a single field. Only clears the changed field's error if now valid (does not add new errors). */
  setField: (
    name: keyof TFormData & string,
    value: TFormData[keyof TFormData]
  ) => void;
  /** Runs full validation and shows all errors. Wire to onBlur. */
  validateAll: () => void;
  /** Returns a form onSubmit handler that calls preventDefault, validates, and gates the callback. */
  handleSubmit: (
    callback: (formData: TFormData) => void | Promise<void>
  ) => (evt: React.FormEvent<HTMLFormElement>) => void;
  /** Clears all shown errors. */
  clearErrors: () => void;
}

// --- Pure validation function ---

function resolveMessage<TFormData>(
  formData: TFormData,
  message?: IValidationMessage<TFormData>
): string | undefined {
  if (message === undefined) {
    return undefined;
  }
  if (typeof message === "string") {
    return message;
  }
  return message(formData);
}

/**
 * Runs validation rules against formData and returns per-field results.
 *
 * Fields are validated in config key insertion order. Each rule's `isValid`
 * receives the in-progress `fields` object, enabling cross-field validation
 * (e.g., field B can check whether field A passed).
 */
export function runValidation<TFormData>(
  formData: TFormData,
  config: IValidationConfig<TFormData>,
  shouldValidateField?: (fieldKey: string, formData: TFormData) => boolean
): IValidationResult {
  const fields: Record<string, IFieldValidation | undefined> = {};
  let isValid = true;

  Object.keys(config).forEach((key) => {
    if (shouldValidateField && !shouldValidateField(key, formData)) {
      return; // skip this field
    }

    const fieldConfig = config[key];
    const failedRule = fieldConfig.validations.find(
      (rule) => !rule.isValid(formData, fields)
    );

    if (failedRule) {
      isValid = false;
      fields[key] = {
        isValid: false,
        message: resolveMessage(formData, failedRule.message),
      };
    } else {
      fields[key] = { isValid: true };
    }
  });

  return { isValid, fields };
}

// --- Hook ---

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useFormValidation<TFormData extends Record<string, any>>({
  initialFormData,
  validationConfig,
  shouldValidateField,
}: IUseFormValidationOptions<TFormData>): IUseFormValidationReturn<TFormData> {
  const [formData, setFormDataState] = useState<TFormData>(initialFormData);
  const [shownErrors, setShownErrors] = useState<
    Record<string, IFieldValidation | undefined>
  >({});

  // Refs prevent stale closures in stable callbacks
  const formDataRef = useRef(formData);
  formDataRef.current = formData;

  const validationConfigRef = useRef(validationConfig);
  const shouldValidateFieldRef = useRef(shouldValidateField);
  shouldValidateFieldRef.current = shouldValidateField;

  // Re-validate when validationConfig changes: clear errors that resolved, keep others
  useEffect(() => {
    validationConfigRef.current = validationConfig;

    const result = runValidation(
      formDataRef.current,
      validationConfig,
      shouldValidateFieldRef.current
    );

    setShownErrors((prev) => {
      let changed = false;
      const next: Record<string, IFieldValidation | undefined> = {};

      Object.keys(prev).forEach((key) => {
        if (prev[key] && !prev[key]?.isValid && result.fields[key]?.isValid) {
          // Error resolved under new config — clear it
          changed = true;
        } else {
          next[key] = prev[key];
        }
      });

      return changed ? next : prev;
    });
  }, [validationConfig]);

  // Derive isValid from shownErrors (not full validation).
  // Initially true (no shown errors). After blur/submit, reflects actual state.
  const isValid = Object.keys(shownErrors).every(
    (key) => !shownErrors[key] || shownErrors[key]?.isValid
  );

  const showAllErrors = useCallback(
    (data: TFormData): IValidationResult => {
      const result = runValidation(
        data,
        validationConfigRef.current,
        shouldValidateFieldRef.current
      );

      setShownErrors(result.fields);

      return result;
    },
    [] // stable: reads from refs
  );

  const setField = useCallback(
    (name: keyof TFormData & string, value: TFormData[keyof TFormData]) => {
      const newFormData = { ...formDataRef.current, [name]: value };
      formDataRef.current = newFormData;
      setFormDataState(newFormData);

      const result = runValidation(
        newFormData,
        validationConfigRef.current,
        shouldValidateFieldRef.current
      );

      setShownErrors((prev) => {
        // Only clear the error for `name` if it was previously shown and is now valid
        if (
          prev[name] &&
          !prev[name]?.isValid &&
          result.fields[name]?.isValid
        ) {
          const next = { ...prev };
          delete next[name];
          return next;
        }
        return prev;
      });
    },
    []
  );

  const setFormData = useCallback(
    (dataOrUpdater: TFormData | ((prev: TFormData) => TFormData)) => {
      const newData =
        typeof dataOrUpdater === "function"
          ? (dataOrUpdater as (prev: TFormData) => TFormData)(
              formDataRef.current
            )
          : dataOrUpdater;

      formDataRef.current = newData;
      setFormDataState(newData);
      showAllErrors(newData);
    },
    [showAllErrors]
  );

  const validateAll = useCallback(() => {
    showAllErrors(formDataRef.current);
  }, [showAllErrors]);

  const handleSubmit = useCallback(
    (callback: (data: TFormData) => void | Promise<void>) => {
      return (evt: React.FormEvent<HTMLFormElement>) => {
        evt.preventDefault();

        const currentData = formDataRef.current;
        const result = showAllErrors(currentData);

        if (result.isValid) {
          callback(currentData);
        }
      };
    },
    [showAllErrors]
  );

  const getFieldError = useCallback(
    (fieldName: string): string | undefined => {
      const field = shownErrors[fieldName];
      if (field && !field.isValid) {
        return field.message;
      }
      return undefined;
    },
    [shownErrors]
  );

  const clearErrors = useCallback(() => {
    setShownErrors({});
  }, []);

  return {
    formData,
    setFormData,
    isValid,
    getFieldError,
    setField,
    validateAll,
    handleSubmit,
    clearErrors,
  };
}
