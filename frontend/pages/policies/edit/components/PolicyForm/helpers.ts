import { getErrorReason } from "interfaces/errors";

// eslint-disable-next-line import/prefer-default-export
export const getPolicyAutomationErrorMessage = (err: unknown) => {
  const declarationSelectedError = getErrorReason(err, {
    reasonIncludes: "resend declaration (DDM)",
  });
  if (declarationSelectedError !== "") {
    return declarationSelectedError;
  }

  const invalidProfilePrefixError = getErrorReason(err, {
    reasonIncludes: "has an invalid prefix",
  });
  if (invalidProfilePrefixError !== "") {
    return "Only Apple and Windows configuration profiles are supported. Please select a valid profile.";
  }

  return "Could not update policy automations.";
};
