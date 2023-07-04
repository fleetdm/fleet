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

export const STYLIZATIONS_AND_ACRONYMS = [
  "macOS",
  "osquery",
  "MySQL",
  "MDM",
  "REST",
  "API",
  "JSON",
];

// fleetdm.com/handbook/marketing/content-style-guide#sentence-case
// * doesn't recognize proper nouns!
export const enforceFleetSentenceCasing = (s: string) => {
  const resArr = s.split(" ").map((word, i) => {
    if (!STYLIZATIONS_AND_ACRONYMS.includes(word)) {
      const lowered = word.toLowerCase();
      if (i === 0) {
        // title case the first word
        return lowered[0].toUpperCase() + lowered.slice(1);
      }
      return lowered;
    }
    return word;
  });

  return resArr.join(" ").trim();
};
export default {
  capitalize,
  capitalizeRole,
};
