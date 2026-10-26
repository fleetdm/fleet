import { AxiosResponse } from "axios";
import React from "react";

import CustomLink from "components/CustomLink";
import { EditorMode } from "components/Editor/Editor";
import { LabelTargetMode, TargetType } from "components/TargetLabelSelector";
import { IApiError } from "interfaces/errors";
import { IMdmProfile, ProfilePlatform } from "interfaces/mdm";
import { generateSecretErrMsg } from "pages/SoftwarePage/helpers";
import { listNamesFromSelectedLabels } from "services/entities/labels";
import { isDDMProfile } from "services/entities/mdm";
import { generateGenericLearnMoreErrMsg } from "utilities/helpers";

export interface IParseFileResult {
  name: string;
  platform: string;
  ext: string;
}

export const parseFile = async (file: File): Promise<IParseFileResult> => {
  // get the file name and extension
  const nameParts = file.name.split(".");
  const name = nameParts.slice(0, -1).join(".");
  const ext = nameParts.slice(-1)[0].toLowerCase();

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

/** The kind of profile a piece of text is, as far as the UI can tell. The
 * server does the real validation; this only picks the file extension the
 * multipart upload needs and the editor's syntax mode. */
export type ProfileContentType =
  | "mobileconfig"
  | "declaration"
  | "android"
  | "windows";

interface IProfileContentTypeInfo {
  /** Shown above the editor. */
  label: string;
  /** Of the file the upload is sent as, which is how the server routes it. */
  extension: string;
  platform: ProfilePlatform;
  editorMode: EditorMode;
}

export const PROFILE_CONTENT_TYPES: Record<
  ProfileContentType,
  IProfileContentTypeInfo
> = {
  mobileconfig: {
    label: "Mobileconfig",
    extension: "mobileconfig",
    platform: "darwin",
    editorMode: "xml",
  },
  declaration: {
    label: "Declaration (DDM)",
    extension: "json",
    platform: "darwin",
    editorMode: "json",
  },
  android: {
    label: "Android",
    extension: "json",
    platform: "android",
    editorMode: "json",
  },
  windows: {
    label: "Windows",
    extension: "xml",
    platform: "windows",
    editorMode: "xml",
  },
};

const SERVER_SECRET_PREFIX = "FLEET_SECRET_";

/** What the file picker offers when adding, where the type isn't known yet. */
export const ADD_PROFILE_ACCEPT =
  ".json,.mobileconfig,application/x-apple-aspen-config,.xml";

const startsUpper = (key: string) =>
  key.charAt(0) !== key.charAt(0).toLowerCase();
const startsLower = (key: string) =>
  key.charAt(0) !== key.charAt(0).toUpperCase();

/** Detects the profile type from pasted text. JSON is a declaration when its
 * top-level keys start uppercase, Android when they start lowercase; XML is a
 * mobileconfig when it is a plist, otherwise Windows SyncML. Returns null for
 * anything else. */
export const detectProfileContentType = (
  text: string
): ProfileContentType | null => {
  const trimmed = text.trim();
  if (trimmed.startsWith("{")) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed === null || typeof parsed !== "object") {
        return null;
      }
      // the server's rule: top-level key casing decides the type
      const keys = Object.keys(parsed);
      const upper = keys.some(startsUpper);
      const lower = keys.some(startsLower);
      if (upper && !lower) {
        return "declaration";
      }
      if (lower && !upper) {
        return "android";
      }
      // the server rejects mixed keys either way; guess so its error shows
      return "Type" in parsed ? "declaration" : "android";
    } catch {
      // the server reads unparseable JSON with a secret as a declaration
      return trimmed.includes(SERVER_SECRET_PREFIX) ? "declaration" : null;
    }
  }
  if (/<plist[\s>]|<!DOCTYPE plist/i.test(trimmed)) {
    return "mobileconfig";
  }
  // SyncML starts with a command or a comment. Behind an XML declaration it's
  // still Windows, so the server reports that the declaration isn't allowed.
  if (
    /^<(replace|add|atomic|exec|!--)/i.test(
      trimmed.replace(/^<\?xml[^>]*\?>\s*/i, "")
    )
  ) {
    return "windows";
  }
  // as the server does, any other XML declaration is read as a plist
  if (/^<\?xml/i.test(trimmed)) {
    return "mobileconfig";
  }
  return null;
};

const EXTENSION_CONTENT_TYPE: Record<string, ProfileContentType> = {
  mobileconfig: "mobileconfig",
  xml: "windows",
  json: "declaration",
};

/** The type the server gives an uploaded file by its extension, for contents
 * the UI can't classify, such as a lone `$FLEET_SECRET_` placeholder. */
export const contentTypeForExtension = (
  ext: string
): ProfileContentType | null => EXTENSION_CONTENT_TYPE[ext] ?? null;

/** The type of an existing profile, from what the API returns about it. */
export const profileContentTypeFor = (
  profile: IMdmProfile
): ProfileContentType => {
  if (isDDMProfile(profile)) {
    return "declaration";
  }
  switch (profile.platform) {
    case "windows":
      return "windows";
    case "android":
      return "android";
    default:
      return "mobileconfig";
  }
};

/** File extensions a replacement for an existing profile may have. */
export const getAcceptedExtensions = (profile: IMdmProfile) => {
  const type = profileContentTypeFor(profile);
  // a mobileconfig is a plist, which may be saved as .xml
  return type === "mobileconfig"
    ? [".mobileconfig", ".xml"]
    : [`.${PROFILE_CONTENT_TYPES[type].extension}`];
};

/** Name given to pasted content that the admin did not name, so the server
 * derives "New profile" for .xml and .json (a .mobileconfig keeps its
 * PayloadDisplayName). */
export const PASTED_PROFILE_DEFAULT_NAME = "New profile";

/** Names are unique per fleet, so an unnamed pasted profile takes "New
 * profile", then the lowest free "New profile N" from 2. Compared without
 * case, as the database does. */
export const nextPastedProfileName = (takenNames: string[]): string => {
  const taken = new Set(takenNames.map((n) => n.trim().toLowerCase()));
  const base = PASTED_PROFILE_DEFAULT_NAME;
  if (!taken.has(base.toLowerCase())) {
    return base;
  }
  let n = 2;
  while (taken.has(`${base} ${n}`.toLowerCase())) {
    n += 1;
  }
  return `${base} ${n}`;
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

const PROFILE_NAME_TAKEN_REASON =
  "A configuration profile with this name already exists";

/** A name clash, which belongs to the name field rather than the whole form. */
export const isProfileNameTakenError = (err: AxiosResponse<IApiError>) =>
  (err?.data?.errors?.[0]?.reason ?? "").includes(PROFILE_NAME_TAKEN_REASON);

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

  if (apiReason.includes(PROFILE_NAME_TAKEN_REASON)) {
    // the server names the flow itself ("Couldn't add." / "Couldn't edit.")
    return apiReason.startsWith("Couldn't")
      ? apiReason
      : `${couldnt} ${apiReason}`;
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
