import React from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";
import { LabelTargetMode, TargetType } from "components/TargetLabelSelector";
import { listNamesFromSelectedLabels } from "services/entities/labels";

import CustomLink from "components/CustomLink";
import { generateGenericLearnMoreErrMsg } from "utilities/helpers";

export interface IParseFileResult {
  name: string;
  platform: string;
  ext: string;
  /** For .json uploads, whether the contents look like an Apple DDM
   * declaration rather than an Android configuration profile. Always false for
   * other extensions. */
  isAppleDeclaration: boolean;
  /** The declaration's `Identifier`, when the upload is an Apple DDM
   * declaration that declares one. */
  declarationIdentifier?: string;
}

const readFileAsText = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error);
    reader.readAsText(file);
  });

interface IAppleDeclarationDetails {
  isAppleDeclaration: boolean;
  declarationIdentifier?: string;
}

const NOT_A_DECLARATION: IAppleDeclarationDetails = {
  isAppleDeclaration: false,
};

/** Distinguishes an Apple DDM declaration from an Android configuration
 * profile, both of which upload as .json. Follows the backend's
 * `DetermineJSONConfigType` (server/mdm/mdm.go): a declaration is keyed in
 * PascalCase and carries "Type" and "Payload", an Android profile uses
 * camelCase keys.
 *
 * This only decides whether to offer the DDM-only activation UI -- the backend
 * is still the authority on whether the file is valid, so contents we can't
 * make sense of are reported as "not a declaration" rather than throwing. */
const parseAppleDeclaration = async (
  file: File
): Promise<IAppleDeclarationDetails> => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(await readFileAsText(file));
  } catch {
    return NOT_A_DECLARATION;
  }

  // `typeof [] === "object"`, so arrays need excluding explicitly.
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return NOT_A_DECLARATION;
  }

  const contents = parsed as Record<string, unknown>;

  // a declaration always names an Apple type; Android profiles have no
  // equivalent key, so this is a positive signal rather than an inference from
  // what the other keys look like.
  if (
    typeof contents.Type !== "string" ||
    !contents.Type.startsWith("com.apple.")
  ) {
    return NOT_A_DECLARATION;
  }

  if (typeof contents.Payload !== "object" || contents.Payload === null) {
    return NOT_A_DECLARATION;
  }

  // Android's camelCase keys are the backend's discriminator, so a lower-case
  // top-level key means this won't be accepted as a declaration.
  if (Object.keys(contents).some((key) => /^[a-z]/.test(key))) {
    return NOT_A_DECLARATION;
  }

  return {
    isAppleDeclaration: true,
    declarationIdentifier:
      typeof contents.Identifier === "string" ? contents.Identifier : undefined,
  };
};

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
        isAppleDeclaration: false,
      };
    }
    case "mobileconfig": {
      return {
        name,
        platform: "macOS, iOS, iPadOS",
        ext,
        isAppleDeclaration: false,
      };
    }
    case "json": {
      return {
        name,
        platform: "Android or macOS(DDM)",
        ext,
        ...(await parseAppleDeclaration(file)),
      };
    }
    default: {
      throw new Error(`Invalid file type: ${ext}`);
    }
  }
};

/** The example activation shown in the custom activation editor, matching the
 * design. Its identifiers are illustrative: `StandardConfigurations` has to
 * name the uploaded declaration's identifier, so this example is a scaffold to
 * edit rather than something that would be accepted as-is.
 *
 * Built via JSON.stringify so it can't drift into invalid JSON. */
export const EXAMPLE_CUSTOM_ACTIVATION = JSON.stringify(
  {
    Type: "com.apple.activation.simple",
    Identifier: "myIdentifier",
    Payload: {
      StandardConfigurations: ["myConfigurationIdentifier"],
    },
  },
  null,
  2
);

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

/** We want to add some additional messaging to some of the error messages so
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
          and start with an uppercase letter. Keys in Android profile must
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
