import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";

import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";

import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "windows-target-form";

interface IWindowsTargetFormData {
  deadlineDays: string;
  gracePeriodDays: string;
}

interface IWindowsTargetFormErrors {
  deadlineDays?: string;
  gracePeriodDays?: string;
}

// validates that a string is a number from 0 to 30
const validateDeadlineDays = (value: string) => {
  if (value === "") return false;

  const parsedValue = Number(value);
  return Number.isInteger(parsedValue) && parsedValue >= 0 && parsedValue <= 30;
};

// validates string is a number from 0 to 7
const validateGracePeriodDays = (value: string) => {
  if (value === "") return false;

  const parsedValue = Number(value);
  return Number.isInteger(parsedValue) && parsedValue >= 0 && parsedValue <= 7;
};

const validateForm = (formData: IWindowsTargetFormData) => {
  const errors: IWindowsTargetFormErrors = {};
  const deadlineEmpty = formData.deadlineDays.trim() === "";
  const graceEmpty = formData.gracePeriodDays.trim() === "";

  // Both empty is valid (clears enforcement)
  if (deadlineEmpty && graceEmpty) {
    return errors;
  }

  if (!deadlineEmpty && !validateDeadlineDays(formData.deadlineDays)) {
    errors.deadlineDays = "Deadline must meet criteria below.";
  }

  if (deadlineEmpty && !graceEmpty) {
    errors.gracePeriodDays =
      "Grace period must be empty if no deadline is set.";
  } else if (!deadlineEmpty && graceEmpty) {
    errors.gracePeriodDays = "The grace period days is required.";
  } else if (!validateGracePeriodDays(formData.gracePeriodDays)) {
    errors.gracePeriodDays = "Grace period must meet criteria below.";
  }

  return errors;
};

interface IWindowsMdmConfigData {
  mdm: {
    windows_updates: {
      deadline_days: number | null;
      grace_period_days: number | null;
    };
  };
}

const createMdmConfigData = (
  deadlineDays: string,
  gracePeriodDays: string
): IWindowsMdmConfigData => {
  return {
    mdm: {
      windows_updates: {
        deadline_days:
          deadlineDays.trim() === "" ? null : parseInt(deadlineDays, 10),
        grace_period_days:
          gracePeriodDays.trim() === "" ? null : parseInt(gracePeriodDays, 10),
      },
    },
  };
};

interface IWindowsTargetFormProps {
  currentTeamId: number;
  defaultDeadlineDays: string;
  defaultGracePeriodDays: string;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
}

const WindowsTargetForm = ({
  currentTeamId,
  defaultDeadlineDays,
  defaultGracePeriodDays,
  refetchAppConfig,
  refetchTeamConfig,
}: IWindowsTargetFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isSaving, setIsSaving] = useState(false);
  const [formData, setFormData] = useState<IWindowsTargetFormData>({
    deadlineDays: defaultDeadlineDays.toString(),
    gracePeriodDays: defaultGracePeriodDays.toString(),
  });
  const [formErrors, setFormErrors] = useState<IWindowsTargetFormErrors>({});

  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while the form is
  // submitting because this component is unmounted.
  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validateForm(formData);
    if (!isEmpty(errors)) {
      setFormErrors(errors);
      return;
    }

    setIsSaving(true);
    const updateData = createMdmConfigData(
      formData.deadlineDays,
      formData.gracePeriodDays
    );
    try {
      currentTeamId === APP_CONTEXT_NO_TEAM_ID
        ? await configAPI.update(updateData)
        : await teamsAPI.update(updateData, currentTeamId);
      renderFlash("success", "Successfully updated Windows OS update options.");
    } catch {
      renderFlash("error", "Couldn’t update. Please try again.");
    } finally {
      currentTeamId === APP_CONTEXT_NO_TEAM_ID
        ? refetchAppConfig()
        : refetchTeamConfig();
      setIsSaving(false);
    }
  };

  const handleChange = (field: keyof IWindowsTargetFormData) => (
    val: string
  ) => {
    const newFormData = { ...formData, [field]: val };
    setFormData(newFormData);
    // On change, only update/clear existing errors (optimistic UX)
    const newErrors = validateForm(newFormData);
    const updatedErrors: IWindowsTargetFormErrors = {};
    Object.keys(formErrors).forEach((key) => {
      const k = key as keyof IWindowsTargetFormErrors;
      if (newErrors[k]) {
        updatedErrors[k] = newErrors[k];
      }
    });
    setFormErrors(updatedErrors);
  };

  const handleBlur = () => {
    setFormErrors(validateForm(formData));
  };

  return (
    <form className={baseClass} onSubmit={handleSubmit}>
      <InputField
        disabled={gitOpsModeEnabled}
        label="Deadline"
        tooltip="Number of days the end user has before updates are installed and the host is forced to restart."
        helpText="Number of days from 0 to 30."
        value={formData.deadlineDays}
        error={formErrors.deadlineDays}
        onChange={handleChange("deadlineDays")}
        onBlur={handleBlur}
      />
      <InputField
        disabled={gitOpsModeEnabled}
        label="Grace period"
        tooltip="Number of days after the deadline the end user has before the host is forced to restart (only if end user was offline when deadline passed)."
        helpText="Number of days from 0 to 7."
        value={formData.gracePeriodDays}
        error={formErrors.gracePeriodDays}
        onChange={handleChange("gracePeriodDays")}
        onBlur={handleBlur}
      />{" "}
      <div className="button-wrap">
        <GitOpsModeTooltipWrapper
          position="right"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              type="submit"
              isLoading={isSaving}
            >
              Save
            </Button>
          )}
        />
      </div>
    </form>
  );
};

export default WindowsTargetForm;
