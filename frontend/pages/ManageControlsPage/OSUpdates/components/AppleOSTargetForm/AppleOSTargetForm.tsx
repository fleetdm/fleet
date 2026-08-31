import React, { useContext, useState } from "react";
import { isEmpty } from "lodash";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { notify } from "components/ToastNotification";
import configAPI from "services/entities/config";
import teamsAPI from "services/entities/teams";
import { ApplePlatform } from "interfaces/platform";

import InputField from "components/forms/fields/InputField";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import validatePresence from "components/forms/validators/validate_presence";
import CustomLink from "components/CustomLink";
import { AppContext } from "context/app";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { getErrorMessage } from "./helpers";

const baseClass = "apple-os-target-form";

/** The sentinel stored in minimum_version meaning "enforce the newest version
 * available", with the deadline derived from deadline_days rather than a date. */
export const LATEST_VERSION = "latest";

/** Which set of fields the form collects. Not stored separately: minimum_version
 * carries the mode, so the target is derived from it. */
export type AppleOSTarget = "none" | "custom" | "latest";

const TARGET_OPTIONS: CustomOptionType[] = [
  { label: "No updates enforced", value: "none" },
  { label: "Custom version", value: "custom" },
  { label: "Latest version", value: "latest" },
];

export const getTargetFromMinOsVersion = (
  minOsVersion: string
): AppleOSTarget => {
  if (minOsVersion === LATEST_VERSION) return "latest";
  return minOsVersion ? "custom" : "none";
};

interface IAppleOSTargetFormData {
  target: AppleOSTarget;
  minOsVersion: string;
  deadline: string;
  deadlineDays: string;
}

interface IAppleOSTargetFormErrors {
  minOsVersion?: string;
  deadline?: string;
  deadlineDays?: string;
}

const validateMinVersion = (value: string) => {
  return /^(0|[1-9]\d*)(\.(0|[1-9]\d*)){0,2}$/.test(value);
};

const validateDeadline = (value: string) => {
  return /^\d{4}-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$/.test(value);
};

/** Whole days only, and there's no upper bound. A deadline of zero days would
 * leave no time to install the update, so 1 is the lowest meaningful value. */
const validateDeadlineDays = (value: string) => {
  return /^[1-9]\d*$/.test(value);
};

const validateForm = (formData: IAppleOSTargetFormData) => {
  const errors: IAppleOSTargetFormErrors = {};

  // Nothing to validate: saving clears the version and the deadline.
  if (formData.target === "none") {
    return errors;
  }

  // Only the days field is shown in "latest" mode, so it's the only one that
  // can be at fault — validating the hidden fields would block the save on an
  // error the user can't see.
  if (formData.target === "latest") {
    if (!validatePresence(formData.deadlineDays)) {
      errors.deadlineDays = "The days after release is required.";
    } else if (!validateDeadlineDays(formData.deadlineDays)) {
      errors.deadlineDays =
        "Days after release must be a whole number of 1 or more.";
    }
    return errors;
  }

  // Both fields are required for a custom version: "No updates enforced" is
  // how enforcement is turned off, so an empty form here isn't a way to clear
  // the settings.
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

interface IAppleOSUpdatesFields {
  minimum_version: string;
  deadline: string;
  deadline_days: number | null;
  update_new_hosts?: boolean;
}

interface IAppleUpdatesMdmConfigData {
  mdm: {
    macos_updates?: IAppleOSUpdatesFields;
    ipados_updates?: IAppleOSUpdatesFields;
    ios_updates?: IAppleOSUpdatesFields;
  };
}

/** Every field is sent for every target, including nulls: the config PATCH
 * merges field by field, so an omitted key leaves the stored value in place. */
const createAppleOSUpdatesData = (
  applePlatform: ApplePlatform,
  formData: IAppleOSTargetFormData,
  updateNewHosts: boolean
): IAppleUpdatesMdmConfigData => {
  const { target, minOsVersion, deadline, deadlineDays } = formData;

  let fields: IAppleOSUpdatesFields;
  switch (target) {
    case "latest":
      fields = {
        minimum_version: LATEST_VERSION,
        // A deadline can't coexist with "latest"; deadline_days replaces it.
        deadline: "",
        deadline_days: parseInt(deadlineDays, 10),
      };
      break;
    case "custom":
      fields = {
        minimum_version: minOsVersion,
        deadline,
        deadline_days: null,
      };
      break;
    default:
      fields = { minimum_version: "", deadline: "", deadline_days: null };
  }

  return {
    mdm: {
      [APPLE_PLATFORMS_TO_CONFIG_FIELDS[applePlatform]]: {
        ...fields,
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
  defaultDeadlineDays: string;
  defaultUpdateNewHosts?: boolean;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
}

const AppleOSTargetForm = ({
  currentTeamId,
  applePlatform,
  defaultMinOsVersion,
  defaultDeadline,
  defaultDeadlineDays,
  defaultUpdateNewHosts,
  refetchAppConfig,
  refetchTeamConfig,
}: IAppleOSTargetFormProps) => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [isSaving, setIsSaving] = useState(false);
  const [target, setTarget] = useState<AppleOSTarget>(
    getTargetFromMinOsVersion(defaultMinOsVersion)
  );
  const [minOsVersion, setMinOsVersion] = useState(
    // The sentinel is a mode, not a version to show in the input.
    defaultMinOsVersion === LATEST_VERSION ? "" : defaultMinOsVersion
  );
  const [deadline, setDeadline] = useState(defaultDeadline);
  const [deadlineDays, setDeadlineDays] = useState(defaultDeadlineDays);
  const [minOsVersionError, setMinOsVersionError] = useState<
    string | undefined
  >();
  const [updateNewHosts, setUpdateNewHosts] = useState<boolean>(
    defaultUpdateNewHosts || false
  );
  const [deadlineError, setDeadlineError] = useState<string | undefined>();
  const [deadlineDaysError, setDeadlineDaysError] = useState<
    string | undefined
  >();

  // "Latest version" always updates new hosts; the other targets leave it to
  // the user. Derived rather than forced into state, and shared with the payload
  // so what's saved matches what the checkbox shows.
  const effectiveUpdateNewHosts = target === "latest" ? true : updateNewHosts;

  // FIXME: This behaves unexpectedly when a user switches tabs or changes the teams dropdown while the form is
  // submitting because this component is unmounted.
  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = { target, minOsVersion, deadline, deadlineDays };
    const errors = validateForm(formData);

    setMinOsVersionError(errors.minOsVersion);
    setDeadlineError(errors.deadline);
    setDeadlineDaysError(errors.deadlineDays);

    if (isEmpty(errors)) {
      setIsSaving(true);
      const updateData = createAppleOSUpdatesData(
        applePlatform,
        formData,
        effectiveUpdateNewHosts
      );
      try {
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? await configAPI.update(updateData)
          : await teamsAPI.update(updateData, currentTeamId);
        notify.success("Successfully updated.");
        // Only refetch on success: the refetch flips isFetching, which unmounts
        // this form behind TargetSection's spinner and remounts it against the
        // stored config -- that's what resets it. After a rejected save nothing
        // changed on the server, so resetting would just discard what the user
        // needs to fix.
        currentTeamId === APP_CONTEXT_NO_TEAM_ID
          ? refetchAppConfig()
          : refetchTeamConfig();
      } catch (err) {
        notify.error(getErrorMessage(err as AxiosResponse<IApiError>), {
          response: err,
        });
      } finally {
        setIsSaving(false);
      }
    }
  };

  const handleTargetChange = (option: CustomOptionType | null) => {
    if (!option) return;
    setTarget(option.value as AppleOSTarget);
    // "Latest version" forces the checkbox on, which isn't a choice the user
    // made, so send it back to the persisted value rather than carrying the
    // previous selection over. The text inputs keep whatever was typed.
    setUpdateNewHosts(defaultUpdateNewHosts || false);
    // The fields these belong to are about to unmount, and a stale message
    // would reappear if the user came back to this target.
    setMinOsVersionError(undefined);
    setDeadlineError(undefined);
    setDeadlineDaysError(undefined);
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
      <DropdownWrapper
        name="target"
        options={TARGET_OPTIONS}
        value={target}
        isDisabled={gitOpsModeEnabled}
        onChange={handleTargetChange}
        helpText={
          target === "latest" ? (
            <>
              Based on host hardware.{" "}
              <CustomLink
                text="Learn more"
                newTab
                url="https://fleetdm.com/learn-more-about/apple-available-os-updates"
              />
            </>
          ) : undefined
        }
      />
      {target === "custom" && (
        <>
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
            tooltip="The end user can't dismiss the OS update once they reach this deadline. Deadline is 12:00 (Noon), the host's local time."
            helpText="YYYY-MM-DD format only (e.g., “2024-07-01”)."
            value={deadline}
            error={deadlineError}
            onChange={handleDeadlineChange}
          />
        </>
      )}
      {target === "latest" && (
        <InputField
          disabled={gitOpsModeEnabled}
          name="deadline_days"
          label="Days after release"
          // Deliberately a text input: a number input's native min/step would
          // block submission before validateForm runs, so the user would get a
          // browser tooltip instead of the form's own error styling.
          helpText="Whole number of days, 1 or more."
          tooltip="The number of days after Apple releases an update before hosts are required to install it."
          value={deadlineDays}
          error={deadlineDaysError}
          onChange={setDeadlineDays}
        />
      )}
      {applePlatform === "darwin" && (
        <Checkbox
          name="update_new_hosts"
          // "Latest version" always updates new hosts, so the choice is made
          // for the user rather than hidden from them.
          disabled={gitOpsModeEnabled || target === "latest"}
          onChange={setUpdateNewHosts}
          value={effectiveUpdateNewHosts}
          className={`${baseClass}__checkbox`}
          labelTooltipContent={
            target === "latest"
              ? "During automated enrollment (ADE), all hosts will be updated to latest macOS version."
              : "During automated enrollment (ADE), hosts below the minimum version are updated to the latest version. If a minimum version isn't set, all hosts are updated to the latest version."
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
