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

/**
 * Pluralizes a word based on the entitiy count and the desired suffixes. If no
 * suffixes are provided, the default suffix "s" is used.
 *
 * @param count The number of items.
 * @param root The root of the word, omitting any suffixs.
 * @param pluralSuffix The suffix to add to the root if the count is not 1.
 * @param singularSuffix The suffix to add to the root if the count is 1.
 * @returns A string with the root and the appropriate suffix.
 *
 * @example
 * pluralize(1, "hero", "es", "") // "hero"
 * pluralize(0, "hero", "es", "") // "heroes"
 * pluralize(1, "fair", "ies", "y") // "fairy"
 * pluralize(2, "fair", "ies", "y") // "fairies"
 * pluralize(1, "dragon") // "dragon"
 * pluralize(2, "dragon") // "dragons"
 */
export const pluralize = (
  count: number,
  root: string,
  pluralSuffix = "s",
  singularSuffix = ""
) => {
  return `${root}${count !== 1 ? pluralSuffix : singularSuffix}`;
};

export default {
  capitalize,
  capitalizeRole,
  pluralize,
};
