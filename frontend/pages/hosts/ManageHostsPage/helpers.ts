import { getErrorReason } from "interfaces/errors";

export const isAcceptableStatus = (filter: string): boolean => {
  return (
    filter === "new" ||
    filter === "online" ||
    filter === "offline" ||
    filter === "missing"
  );
};

export const isValidPolicyResponse = (filter: string): boolean => {
  return filter === "pass" || filter === "fail";
};

// Performs a grossly oversimplied validation that subject string includes substrings
// that would be expected in a textual encoding of a certificate chain per the PEM spec
// (see https://datatracker.ietf.org/doc/html/rfc7468#section-2)
// Consider using a third-party library if more robust validation is desired
export const isValidPemCertificate = (cert: string): boolean => {
  const regexPemHeader = /-----BEGIN/;
  const regexPemFooter = /-----END/;

  return regexPemHeader.test(cert) && regexPemFooter.test(cert);
};

const hasStatusKey = (value: unknown): value is { status: number } => {
  return (
    typeof value === "object" &&
    value !== null &&
    "status" in value &&
    typeof (value as any).status === "number"
  );
};

export const getDeleteLabelErrorMessages = (error: unknown): string => {
  // unprocessable content status. Label is used in a custom profile
  // or software target. we have to check that status exists on the error object
  // before we can access it.
  if (hasStatusKey(error) && error.status === 422) {
    return getErrorReason(error).includes("built-in")
      ? "Built-in labels can't be modified or deleted."
      : "Couldn't delete. Software uses this label as a custom target. Remove the label from the software target and try again.";
  }

  return "Could not delete label. Please try again.";
};
