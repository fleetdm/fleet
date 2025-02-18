import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";

import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { NotificationContext } from "context/notification";
import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";

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

  if (!validatePresence(formData.deadlineDays)) {
    errors.deadlineDays = "The deadline days is required.";
  } else if (!validateDeadlineDays(formData.deadlineDays)) {
    errors.deadlineDays = "Deadline must meet criteria below.";
  }

  if (!validatePresence(formData.gracePeriodDays)) {
    errors.gracePeriodDays = "The grace period days is required.";
  } else if (!validateGracePeriodDays(formData.gracePeriodDays)) {
    errors.gracePeriodDays = "Grace period must meet criteria below.";
  }

  return errors;
};

interface IWindowsMdmConfigData {
  mdm: {
    windows_updates: {
      deadline_days: number;
      grace_period_days: number;
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
        deadline_days: parseInt(deadlineDays, 10),
        grace_period_days: parseInt(gracePeriodDays, 10),
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
  const [isSaving, setIsSaving] = useState(false);
  const [deadlineDays, setDeadlineDays] = useState(
    defaultDeadlineDays.toString()
  );
  const [gracePeriodDays, setGracePeriodDays] = useState(
    defaultGracePeriodDays.toString()
  );
  const [deadlineDaysError, setDeadlineDaysError] = useState<
    string | undefined
  >();
  const [gracePeriodDaysError, setGracePeriodDaysError] = useState<
    string | undefined
  >();

  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while the form is
  // submitting because this component is unmounted.
  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validateForm({
      deadlineDays,
      gracePeriodDays,
    });

    setDeadlineDaysError(errors.deadlineDays);
    setGracePeriodDaysError(errors.gracePeriodDays);

    if (isEmpty(errors)) {
      setIsSaving(true);
      const updateData = createMdmConfigData(deadlineDays, gracePeriodDays);
      try {
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? await configAPI.update(updateData)
          : await teamsAPI.update(updateData, currentTeamId);
        renderFlash(
          "success",
          "Successfully updated Windows OS update options."
        );
      } catch {
        renderFlash("error", "Couldn’t update. Please try again.");
      } finally {
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? refetchAppConfig()
          : refetchTeamConfig();
        setIsSaving(false);
      }
    }
  };

  const handleDeadlineDaysChange = (val: string) => {
    setDeadlineDays(val);
  };

  const handleGracePeriodDays = (val: string) => {
    setGracePeriodDays(val);
  };

  return (
    <form className={baseClass} onSubmit={handleSubmit}>
      <InputField
        label="Deadline"
        tooltip="Number of days the end user has before updates are installed and the host is forced to restart."
        helpText="Number of days from 0 to 30."
        value={deadlineDays}
        error={deadlineDaysError}
        onChange={handleDeadlineDaysChange}
      />
      <InputField
        label="Grace period"
        tooltip="Number of days after the deadline the end user has before the host is forced to restart (only if end user was offline when deadline passed)."
        helpText="Number of days from 0 to 7."
        value={gracePeriodDays}
        error={gracePeriodDaysError}
        onChange={handleGracePeriodDays}
      />
      <Button type="submit" isLoading={isSaving}>
        Save
      </Button>
    </form>
  );
};

export default WindowsTargetForm;
