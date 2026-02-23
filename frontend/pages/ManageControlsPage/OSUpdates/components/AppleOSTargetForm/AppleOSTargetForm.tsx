import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";

import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { NotificationContext } from "context/notification";
import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";
import { ApplePlatform } from "interfaces/platform";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import validatePresence from "components/forms/validators/validate_presence";
import CustomLink from "components/CustomLink";
import { AppContext } from "context/app";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "apple-os-target-form";

interface IAppleOSTargetFormData {
  minOsVersion: string;
  deadline: string;
}

interface IAppleOSTargetFormErrors {
  minOsVersion?: string;
  deadline?: string;
}

const validateMinVersion = (value: string) => {
  return /^(0|[1-9]\d*)(\.(0|[1-9]\d*)){0,2}$/.test(value);
};

const validateDeadline = (value: string) => {
  return /^\d{4}-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$/.test(value);
};

const validateForm = (formData: IAppleOSTargetFormData) => {
  const errors: IAppleOSTargetFormErrors = {};

  // Both fields may be cleared out and saved
  if (
    !validatePresence(formData.minOsVersion) &&
    !validatePresence(formData.deadline)
  ) {
    return errors;
  }

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

const APPLE_PLATFORMS_TO_CONFIG_FIELDS = {
  darwin: "macos_updates",
  ios: "ios_updates",
  ipados: "ipados_updates",
};

interface IAppleUpdatesMdmConfigData {
  mdm: {
    macos_updates?: {
      minimum_version: string;
      deadline: string;
    };
    ipados_updates?: {
      minimum_version: string;
      deadline: string;
    };
    ios_updates?: {
      minimum_version: string;
      deadline: string;
    };
  };
}

const createAppleOSUpdatesData = (
  applePlatform: ApplePlatform,
  minOsVersion: string,
  deadline: string,
  updateNewHosts?: boolean
): IAppleUpdatesMdmConfigData => {
  return {
    mdm: {
      [APPLE_PLATFORMS_TO_CONFIG_FIELDS[applePlatform]]: {
        minimum_version: minOsVersion,
        deadline,
        // Add update_new_hosts only for macOS right now.
        ...(applePlatform === "darwin"
          ? { update_new_hosts: updateNewHosts }
          : {}),
      },
    },
  };
};

interface IAppleOSTargetFormProps {
  currentTeamId: number;
  applePlatform: ApplePlatform;
  defaultMinOsVersion: string;
  defaultDeadline: string;
  defaultUpdateNewHosts?: boolean;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
}

const AppleOSTargetForm = ({
  currentTeamId,
  applePlatform,
  defaultMinOsVersion,
  defaultDeadline,
  defaultUpdateNewHosts,
  refetchAppConfig,
  refetchTeamConfig,
}: IAppleOSTargetFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isSaving, setIsSaving] = useState(false);
  const [minOsVersion, setMinOsVersion] = useState(defaultMinOsVersion);
  const [deadline, setDeadline] = useState(defaultDeadline);
  const [minOsVersionError, setMinOsVersionError] = useState<
    string | undefined
  >();
  const [updateNewHosts, setUpdateNewHosts] = useState<boolean>(
    defaultUpdateNewHosts || false
  );
  const [deadlineError, setDeadlineError] = useState<string | undefined>();

  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while the form is
  // submitting because this component is unmounted.
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
      const updateData = createAppleOSUpdatesData(
        applePlatform,
        minOsVersion,
        deadline,
        updateNewHosts
      );
      try {
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? await configAPI.update(updateData)
          : await teamsAPI.update(updateData, currentTeamId);
        renderFlash("success", "Successfully updated.");
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

  const handleMinVersionChange = (val: string) => {
    setMinOsVersion(val);
  };

  const handleDeadlineChange = (val: string) => {
    setDeadline(val);
  };

  const getMinimumVersionTooltip = () => {
    return <>Enrolled hosts are updated to exactly this version.</>;
  };

  return (
    <form className={baseClass} onSubmit={handleSubmit}>
      <InputField
        label="Minimum version"
        name="minimum_version"
        disabled={gitOpsModeEnabled}
        tooltip={getMinimumVersionTooltip()}
        helpText={
          <>
            Use only versions{" "}
            <CustomLink
              text="available from Apple."
              newTab
              url="https://fleetdm.com/learn-more-about/apple-available-os-updates"
            />
          </>
        }
        value={minOsVersion}
        error={minOsVersionError}
        onChange={handleMinVersionChange}
      />
      <InputField
        disabled={gitOpsModeEnabled}
        name="deadline"
        label="Deadline"
        tooltip="The end user can't dismiss the OS update once they reach this deadline. Deadline is 19:00 (7PM), the host's local time."
        helpText="YYYY-MM-DD format only (e.g., “2024-07-01”)."
        value={deadline}
        error={deadlineError}
        onChange={handleDeadlineChange}
      />
      {applePlatform === "darwin" && (
        <Checkbox
          name="update_new_hosts"
          disabled={gitOpsModeEnabled}
          onChange={setUpdateNewHosts}
          value={updateNewHosts}
          className={`${baseClass}__checkbox`}
          labelTooltipContent={
            "Hosts that automatically enroll (ADE) are updated to Apple's latest version during macOS Setup Assistant."
          }
        >
          Update new hosts to latest
        </Checkbox>
      )}
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

export default AppleOSTargetForm;
