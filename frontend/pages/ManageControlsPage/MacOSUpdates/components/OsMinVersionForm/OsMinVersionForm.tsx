import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { isEmpty } from "lodash";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import validatePresence from "components/forms/validators/validate_presence";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";

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

const createMdmConfigData = (minOsVersion: string, deadline: string) => {
  return {
    mdm: {
      macos_updates: {
        minimum_version: minOsVersion,
        deadline,
      },
    },
  };
};

interface IOsMinVersionForm {
  currentTeamId?: number;
}

const OsMinVersionForm = ({
  currentTeamId = APP_CONTEXT_NO_TEAM_ID,
}: IOsMinVersionForm) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const [isSaving, setIsSaving] = useState(false);
  const [minOsVersion, setMinOsVersion] = useState(
    currentTeamId === APP_CONTEXT_NO_TEAM_ID
      ? config?.mdm.macos_updates.minimum_version ?? ""
      : ""
  );
  const [deadline, setDeadline] = useState(
    currentTeamId === APP_CONTEXT_NO_TEAM_ID
      ? config?.mdm.macos_updates.deadline ?? ""
      : ""
  );
  const [minOsVersionError, setMinOsVersionError] = useState<
    string | undefined
  >();
  const [deadlineError, setDeadlineError] = useState<string | undefined>();

  useQuery<ILoadTeamResponse, Error>(
    ["apple mdm config", currentTeamId],

    // NOTE: this method should never be called with 0 as we sure to have
    // a value for current team from the "enabled" option. We add it here
    // to fulfill correct typing.
    () => teamsAPI.load(currentTeamId || 0),
    {
      refetchOnWindowFocus: false,
      staleTime: Infinity,
      enabled: currentTeamId > APP_CONTEXT_NO_TEAM_ID,
      onSuccess: (data) => {
        setMinOsVersion(data.team?.mdm?.macos_updates?.minimum_version ?? "");
        setDeadline(data.team?.mdm?.macos_updates?.deadline ?? "");
      },
    }
  );

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const errors = validateForm({
      minOsVersion,
      deadline,
    });

    setMinOsVersionError(errors.minOsVersion);
    setDeadlineError(errors.deadline);

    if (isEmpty(errors)) {
      setIsSaving(true);
      const updateData = createMdmConfigData(minOsVersion, deadline);
      try {
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? await configAPI.update(updateData)
          : await teamsAPI.update(updateData, currentTeamId);
        renderFlash("success", "Successfully updated minimum version!");
      } catch {
        renderFlash("error", "Couldn’t update. Please try again.");
      } finally {
        setIsSaving(false);
      }
    }
  };

  const handleMinVersionChange = (val: string) => {
    setMinOsVersion(val);
  };

  const handleDeadlineChange = (val: string) => {
    setDeadline(val);
  };

  return (
    <form className={baseClass} onSubmit={handleSubmit}>
      <InputField
        label="Minimum version"
        tooltip="The end user sees the window until their macOS is at or above this version."
        hint="Version number only (e.g., “13.0.1” not “Ventura 13” or “13.0.1 (22A400)”)"
        placeholder="13.0.1"
        value={minOsVersion}
        error={minOsVersionError}
        onChange={handleMinVersionChange}
      />
      <InputField
        label="Deadline"
        tooltip="The end user can’t dismiss the window once they reach this deadline. Deadline is at 12:00 (Noon) Pacific Standard Time (GMT-8)."
        hint="YYYY-MM-DD format only (e.g., “2023-06-01”)."
        placeholder="2023-06-01"
        value={deadline}
        error={deadlineError}
        onChange={handleDeadlineChange}
      />
      <Button type="submit" isLoading={isSaving}>
        Save
      </Button>
    </form>
  );
};

export default OsMinVersionForm;
