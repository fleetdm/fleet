import React from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";
import { LabelTargetMode, TargetType } from "components/TargetLabelSelector";
import { listNamesFromSelectedLabels } from "services/entities/labels";

import CustomLink from "components/CustomLink";

export interface IParseFileResult {
  name: string;
  platform: string;
  ext: string;
}

export const parseFile = async (file: File): Promise<IParseFileResult> => {
  // get the file name and extension
  const nameParts = file.name.split(".");
  const name = nameParts.slice(0, -1).join(".");
  const ext = nameParts.slice(-1)[0];

  switch (ext) {
    case "xml": {
      return {
        name,
        platform: "Windows",
        ext,
      };
    }
    case "mobileconfig": {
      return { name, platform: "macOS, iOS, iPadOS", ext };
    }
    case "json": {
      return { name, platform: "Android or macOS(DDM)", ext };
    }
    default: {
      throw new Error(`Invalid file type: ${ext}`);
    }
  }
};

interface IGenerateCustomTargetLabelKeyArgs {
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: Record<string, boolean>;
  excludeLabels: Record<string, boolean>;
}

export const generateCustomTargetLabelKey = ({
  targetType,
  includeMode,
  includeLabels,
  excludeLabels,
}: IGenerateCustomTargetLabelKeyArgs) => {
  if (targetType !== "Custom") {
    return {};
  }

  const result: Record<string, string[]> = {};
  const includeNames = listNamesFromSelectedLabels(includeLabels);
  const excludeNames = listNamesFromSelectedLabels(excludeLabels);
  if (includeNames.length) {
    result[
      includeMode === "all" ? "labelsIncludeAll" : "labelsIncludeAny"
    ] = includeNames;
  }
  if (excludeNames.length) {
    result.labelsExcludeAny = excludeNames;
  }
  return result;
};

export const DEFAULT_ERROR_MESSAGE =
  "Couldn't add configuration profile. Please try again.";
export const DEFAULT_EDIT_ERROR_MESSAGE =
  "Couldn't edit configuration profile. Please try again.";

export type ProfileErrorAction = "add" | "edit";

const generateUnsupportedVariableErrMsg = (
  errMsg: string,
  couldnt: string,
  defaultMessage: string
) => {
  const regex = /\$[A-Z0-9_]+/;
  const varName = errMsg.match(regex);
  return varName
    ? `${couldnt} Variable "${varName[0]}" doesn't exist.`
    : defaultMessage;
};

const generateSCEPLearnMoreErrMsg = (
  errMsg: string,
  learnMoreUrl: string,
  couldnt: string
) => {
  return (
    <>
      {couldnt} {errMsg}{" "}
      <CustomLink
        url={learnMoreUrl}
        text="Learn more"
        variant="flash-message-link"
        newTab
      />
    </>
  );
};

/**
 * Helper function to take whatever message is from the API and strip out the Learn More link and format it accordingly.
 */
const generateGenericLearnMoreErrMsg = (errMsg: string) => {
  if (errMsg.includes(" Learn more: https://")) {
    const message = errMsg.substring(
      0,
      errMsg.indexOf(" Learn more: https://")
    );
    const link = errMsg.substring(errMsg.indexOf("https://"));
    return (
      <>
        {message}{" "}
        <CustomLink
          url={link}
          text="Learn more"
          variant="flash-message-link"
          newTab
        />
      </>
    );
  }
  return errMsg;
};

/** We want to add some additional messageing to some of the error messages so
 * we add them in this function. Otherwise, we'll just return the error message from the
 * API. Pass `action: "edit"` when the error came from editing an existing
 * profile so the added messaging reads "Couldn't edit." instead of
 * "Couldn't add.".
 */
export const getErrorMessage = (
  err: AxiosResponse<IApiError>,
  action: ProfileErrorAction = "add"
) => {
  const apiReason = err?.data?.errors?.[0]?.reason ?? "";
  const couldnt = action === "edit" ? "Couldn't edit." : "Couldn't add.";
  const defaultMessage =
    action === "edit" ? DEFAULT_EDIT_ERROR_MESSAGE : DEFAULT_ERROR_MESSAGE;

  if (apiReason.includes("should include valid JSON")) {
    return `${couldnt} The profile should include valid JSON.`;
  }

  if (apiReason.includes("JSON is empty")) {
    return `${couldnt} The JSON file doesn't include any fields.`;
  }

  if (apiReason.includes("Keys in declaration (DDM) profile")) {
    return (
      <div className="upload-profile-invalid-keys-error">
        <span>
          {couldnt} Keys in declaration (DDM) profile must contain only letters
          and start with a uppercase letter. Keys in Android profile must
          contain only letters and start with a lowercase letter.{" "}
        </span>
        <CustomLink
          text="Learn more"
          newTab
          variant="flash-message-link"
          url="https://fleetdm.com/learn-more-about/how-to-craft-android-profile"
        />
      </div>
    );
  }

  if (
    apiReason.includes("apple declaration missing Type") ||
    apiReason.includes("apple declaration missing Payload")
  ) {
    return `${couldnt} Declaration (DDM) profile must include "Type" and "Payload" fields.`;
  }

  if (
    apiReason.includes(
      'Android configuration profile can\'t include "statusReportingSettings"'
    )
  ) {
    return (
      <>
        <span>
          {couldnt} Android configuration profile can&apos;t include
          {'"statusReportingSettings"'} setting. To see host vitals, go to{" "}
          <b>Host details</b>.
        </span>
      </>
    );
  }

  if (
    apiReason.includes(
      "The configuration profile can't include BitLocker settings."
    )
  ) {
    return (
      <span>
        {couldnt} The configuration profile can&apos;t include BitLocker
        settings. To control these settings, go to <b>Disk encryption</b>.
      </span>
    );
  }

  if (
    apiReason.includes(
      "The configuration profile can't include FileVault settings."
    )
  ) {
    return (
      <span>
        {couldnt} The configuration profile can&apos;t include FileVault
        settings. To control these settings, go to <b>Disk encryption</b>.
      </span>
    );
  }

  if (
    apiReason.includes(
      "The configuration profile can't include Windows update settings."
    )
  ) {
    return (
      <span>
        {apiReason} To control these settings, go to <b>OS updates</b>.
      </span>
    );
  }

  // profile mismatch errors only occur on the edit flow (checked before the
  // plain "Identifier" match because it is a substring of "PayloadIdentifier")
  if (
    apiReason.includes(
      "The new profile's PayloadIdentifier must match the existing profile's."
    )
  ) {
    return "Couldn't edit. The uploaded profile must have the same PayloadIdentifier as the original profile.";
  }

  if (
    apiReason.includes(
      "The new profile's Identifier must match the existing profile's."
    )
  ) {
    return "Couldn't edit. The uploaded profile must have the same identifier as the original profile.";
  }

  if (
    apiReason.includes(
      "The new profile's name must match the existing profile's name."
    )
  ) {
    return "Couldn't edit. The uploaded profile must have the same name as the original profile.";
  }

  if (apiReason.includes("OS updates are already configured")) {
    // the backend message is phrased for the add flow ("Couldn't add
    // profile. ..."), so rephrase the prefix for edits.
    return action === "edit"
      ? "Couldn't edit profile. OS updates are already configured. Remove the OS updates settings first."
      : apiReason;
  }

  if (apiReason.includes("Secret variable")) {
    return generateSecretErrMsg(err);
  }

  if (
    apiReason.includes("Fleet variable") &&
    apiReason.includes("not supported in configuration profiles")
  ) {
    return generateUnsupportedVariableErrMsg(
      apiReason,
      couldnt,
      defaultMessage
    );
  }

  if (
    apiReason.includes(
      "can't be used if variables for SCEP URL and Challenge are not specified"
    )
  ) {
    return generateSCEPLearnMoreErrMsg(
      apiReason,
      "https://fleetdm.com/learn-more-about/certificate-authorities",
      couldnt
    );
  }

  if (
    apiReason.includes(
      "SCEP profile for custom SCEP certificate authority requires"
    )
  ) {
    return generateSCEPLearnMoreErrMsg(
      apiReason,
      "https://fleetdm.com/learn-more-about/custom-scep-configuration-profile",
      couldnt
    );
  }

  if (
    apiReason.includes(
      "SCEP profile for NDES certificate authority requires: $FLEET_VAR_NDES_SCEP_CHALLENGE"
    )
  ) {
    return generateSCEPLearnMoreErrMsg(
      apiReason,
      "https://fleetdm.com/learn-more-about/ndes-scep-configuration-profile",
      couldnt
    );
  }

  if (apiReason.includes('"PayloadScope"')) {
    return generateGenericLearnMoreErrMsg(apiReason);
  }

  if (apiReason.includes("Configuration profiles can't be signed")) {
    return generateGenericLearnMoreErrMsg(apiReason);
  }

  // // FIXME: Should we include a default case to catch any other learn more links from the API?
  // // Can we get rid of some/all of the specific cases above and just have this generic one?
  // if (apiReason.includes(" Learn more: https://")) {
  //   return generateGenericLearnMoreErrMsg(apiReason);
  // }

  return apiReason || defaultMessage;
};
