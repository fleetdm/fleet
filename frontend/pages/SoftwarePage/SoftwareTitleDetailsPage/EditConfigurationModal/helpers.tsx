import React from "react";

import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE =
  "Couldn't update configuration. Please try again.";

/** Variables that are valid in configuration profiles but NOT in managed configuration. */
const PROFILE_ONLY_VARIABLE_PATTERNS = [
  /^NDES_SCEP_/,
  /^CUSTOM_SCEP_/,
  /^SCEP_RENEWAL_ID$/,
  /^DIGICERT_/,
  /^SCEP_WINDOWS_CERTIFICATE_ID$/,
  /^SMALLSTEP_SCEP_/,
];

const isProfileOnlyVariable = (varNameWithoutPrefix: string): boolean => {
  return PROFILE_ONLY_VARIABLE_PATTERNS.some((pattern) =>
    pattern.test(varNameWithoutPrefix)
  );
};

const generateUnsupportedVariableErrMsg = (errMsg: string) => {
  const match = errMsg.match(/\$FLEET_VAR_(\w+)/);
  if (!match) {
    return DEFAULT_ERROR_MESSAGE;
  }
  const fullVarName = match[0];
  const varNameWithoutPrefix = match[1];

  if (isProfileOnlyVariable(varNameWithoutPrefix)) {
    return `Couldn't add. Variable "${fullVarName}" isn't supported in managed configuration. It can only be used in configuration profiles.`;
  }

  return `Couldn't add. Variable "${fullVarName}" doesn't exist.`;
};

const generateMissingSecretErrMsg = (errMsg: string) => {
  const matches = [...errMsg.matchAll(/"\$FLEET_SECRET_\w+"/g)];
  if (matches.length === 0) {
    return DEFAULT_ERROR_MESSAGE;
  }
  const varNames = matches.map((m) => m[0].replace(/"/g, ""));
  const plural = varNames.length > 1 ? "s" : "";
  const quoted = varNames.map((v) => `"${v}"`).join(", ");
  return `Couldn't add. Variable${plural} ${quoted} doesn't exist.`;
};

export const getErrorMessage = (err: unknown, isApplePlatform: boolean) => {
  const reason = getErrorReason(err);

  // Android-specific: backend rejects top-level keys other than these two
  if (
    !isApplePlatform &&
    (reason.includes("managedConfiguration") ||
      reason.includes("workProfileWidgets"))
  ) {
    return (
      <>
        Couldn&apos;t update configuration. Only
        &quot;managedConfiguration&quot; and &quot;workProfileWidgets&quot; are
        supported as top-level keys.
      </>
    );
  }

  // Fleet variable unsupported in managed configuration.
  // Note: the backend validates variables one at a time and returns on the
  // first unsupported one it finds, so only one variable is surfaced per
  // request even if the configuration contains multiple invalid variables.
  if (reason.includes("unsupported variable")) {
    return generateUnsupportedVariableErrMsg(reason);
  }

  // Secret variable missing from database
  if (reason.includes("Secret variable")) {
    return generateMissingSecretErrMsg(reason);
  }

  return reason || DEFAULT_ERROR_MESSAGE;
};

// Used to surface error.message in UI of unknown error type
type ErrorWithMessage = {
  message: string;
  [key: string]: unknown;
};

export const isErrorWithMessage = (
  error: unknown
): error is ErrorWithMessage => {
  return (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof (error as ErrorWithMessage).message === "string"
  );
};

/** Validates a JSON string. Returns an error message or null if valid. */
export const validateJson = (value: string): string | null => {
  if (!value) {
    return null;
  }
  try {
    JSON.parse(value);
  } catch (e: unknown) {
    if (isErrorWithMessage(e)) {
      return e.message.toString();
    }
    throw e;
  }
  return null;
};

/** Validates an XML plist string. Returns an error message or null if valid. */
export const validateXml = (value: string): string | null => {
  if (!value) {
    return null;
  }

  const parser = new DOMParser();
  const doc = parser.parseFromString(value, "application/xml");

  const parseError = doc.querySelector("parsererror");
  if (parseError) {
    return "Invalid XML";
  }

  const root = doc.documentElement;

  // Accept <dict> directly
  if (root.tagName === "dict") {
    return null;
  }

  // Accept <plist>-wrapped configs when the plist's root value is a <dict>
  if (root.tagName === "plist" && root.firstElementChild?.tagName === "dict") {
    return null;
  }

  if (root.tagName === "plist") {
    return "<plist> root must contain a <dict> element.";
  }

  return "Root element must be <dict>. Apple managed app configurations require a <dict> root element.";
};

/** Returns display label for a platform value. */
export const getPlatformLabel = (platform: string): string => {
  switch (platform) {
    case "ios":
      return "iOS";
    case "ipados":
      return "iPadOS";
    case "android":
      return "Android";
    default:
      return platform;
  }
};
