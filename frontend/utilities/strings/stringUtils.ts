import { UserRole } from "interfaces/user";

/**
 * Capitalizes the words of the string passed in.
 * @param str un-capitalized string
 */
const capitalize = (str: string): string => {
  return str.replace(/\b\w/g, (letter) => letter.toUpperCase());
};

const capitalizeRole = (str: UserRole): UserRole => {
  if (str === "observer_plus") {
    return "Observer+";
  }
  if (str === "gitops") {
    return "GitOps";
  }
  return str.replace(/\b\w/g, (letter) => letter.toUpperCase()) as UserRole;
};

export default {
  capitalize,
  capitalizeRole,
};
