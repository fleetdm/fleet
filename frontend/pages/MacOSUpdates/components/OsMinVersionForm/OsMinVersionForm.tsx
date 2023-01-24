import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";

import { NotificationContext } from "context/notification";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";

const baseClass = "os-min-version-form";

interface IMinOsVersionFormData {
  minOsVersion: string;
  deadline: string;
}

interface IMinOsVersionFormErrors {
  minOsVersion?: string;
  deadline?: string;
}

const validateMinVersion = (value: string) => {
  return /^(0|[1-9]\d*)(\.(0|[1-9]\d*)){0,2}$/.test(value);
};

const validateDeadline = (value: string) => {
  return /^\d{4}-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$/.test(value);
};

const validateForm = (formData: IMinOsVersionFormData) => {
  const errors: IMinOsVersionFormErrors = {};

  if (!validatePresence(formData.minOsVersion)) {
    errors.minOsVersion = "The minimum version is required.";
  } else if (!validateMinVersion(formData.minOsVersion)) {
    errors.minOsVersion = "Minimum version must meet criteria below.";
  }

  if (!validatePresence(formData.deadline)) {
    errors.deadline = "The deadline is required.";
  } else if (!validateDeadline(formData.deadline)) {
    errors.deadline = "Deadline must meet criteria below.";
  }

  return errors;
};

const OsMinVersionForm = () => {
  const [minOsVersion, setMinOsVersion] = useState(""); // TODO: get default val
  const [deadline, setDeadling] = useState(""); // TODO: get default val
  const [minOsVersionErr, setMinOsVersionErr] = useState<string | undefined>();
  const [deadlineErr, setDeadlineErr] = useState<string | undefined>();

  const { renderFlash } = useContext(NotificationContext);

  const [isSaving, setIsSaving] = useState(false);

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validateForm({
      minOsVersion,
      deadline,
    });

    setMinOsVersionErr(errors.minOsVersion);
    setDeadlineErr(errors.deadline);

    if (isEmpty(errors)) {
      // TODO: request to API
      setIsSaving(true);
      setTimeout(() => {
        renderFlash("success", "Successfully updated minimum version!");
        setIsSaving(false);
      }, 1000);
    }
  };

  const handleMinVersionChange = (val: string) => {
    setMinOsVersion(val);
  };

  const handleDeadlineChange = (val: string) => {
    setDeadling(val);
  };

  return (
    <form className={baseClass} onSubmit={handleSubmit}>
      <InputField
        label="Minimum version"
        tooltip="The end user sees the window until their macOS is at or above this version."
        hint="Version number only (e.g., “13.0.1.”) NOT “Ventura 13” or “13.0.1 (22A400)."
        value={minOsVersion}
        error={minOsVersionErr}
        onChange={handleMinVersionChange}
      />
      <InputField
        label="Deadline"
        tooltip="The end user can’t dismiss the window once they reach this deadline. Deadline is at 12:00 (Noon) Pacific Standard Time (GMT-8)."
        hint="YYYY-MM-DD format only (e.g., “2023-06-01”)."
        value={deadline}
        error={deadlineErr}
        onChange={handleDeadlineChange}
      />
      <Button type="submit" isLoading={isSaving}>
        Save
      </Button>
    </form>
  );
};

export default OsMinVersionForm;
