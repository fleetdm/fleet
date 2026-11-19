import { isFleetApiError } from "interfaces/errors";

export const FLEET_NAME_CONFLICT_ERROR =
  "Name is too similar to another fleet name. Please try another name.";
export const FLEET_NAME_NO_LETTER_OR_NUMBER_ERROR =
  "Fleet name must include at least one letter or number.";

/**
 * getFleetNameError returns the error to show under the "Fleet name" field
 * for a failed create or rename request, or undefined when the failure isn't
 * one the user can fix by changing the name.
 */
export const getFleetNameError = (
  response: unknown,
  name: string
): string | undefined => {
  const { status, data } = (response ?? {}) as {
    status?: unknown;
    data?: { errors?: unknown };
  };

  // The server words conflicts differently per endpoint, so go by status.
  if (status === 409) {
    return FLEET_NAME_CONFLICT_ERROR;
  }

  // Every invalid name is a 422 on "name", so only the reason tells them apart.
  // The body isn't Fleet's when a proxy answers, so check its shape.
  const errors: unknown[] = Array.isArray(data?.errors) ? data.errors : [];
  const nameError = errors
    .filter(isFleetApiError)
    .find((e) => e.name === "name");
  if (status !== 422 || !nameError) {
    return undefined;
  }
  const reason = nameError.reason.toLowerCase();
  if (reason.includes("at least one letter or number")) {
    return FLEET_NAME_NO_LETTER_OR_NUMBER_ERROR;
  }
  // The reason starts with the name the user typed, which can make the error wrap.
  const similarToReserved = nameError.reason.lastIndexOf(
    "is too similar to the reserved name"
  );
  if (similarToReserved !== -1) {
    return `Name ${nameError.reason.slice(
      similarToReserved
    )}. Please try another name.`;
  }
  if (reason.includes("reserved fleet name")) {
    return `"${name}" is a reserved fleet name. Please try another name.`;
  }
  return undefined;
};
