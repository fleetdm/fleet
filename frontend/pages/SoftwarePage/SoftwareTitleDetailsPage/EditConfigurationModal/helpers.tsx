import React from "react";

import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE =
  "Couldn't update configuration. Please try again.";

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
