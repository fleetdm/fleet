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
  "iOS",
  "iPadOS",
  "osquery",
  "MySQL",
  "MDM",
  "REST",
  "API",
  "JSON",
];

// fleetdm.com/handbook/marketing/content-style-guide#sentence-case
/** Does not recognize proper nouns! */
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

export const strToBool = (str?: string | null) => {
  return str ? JSON.parse(str) : false;
};

export const stripQuotes = (string: string) => {
  // Regular expression to match quotes at the start and end of the string
  const quoteRegex = /^([''""])([\s\S]*?)(\1)$/;

  // If the string matches the regex, return the content between the quotes
  // Otherwise, return the original string
  const match = string.match(quoteRegex);
  return match ? match[2] : string;
};

export const isIncompleteQuoteQuery = (str: string) => {
  const pattern = /^(['"])(?!.*\1$)/;
  return pattern.test(str);
};

/**
 * Hyphenates the words of the string passed in.
 * e.g. The name of an app to be used in a file name
 * @param str un-capitalized string
 */
export const hyphenateString = (str: string): string => {
  return str.trim().toLowerCase().replace(/\s+/g, "-");
};

/**
 * This attempts to loosely match the provided string to a key in a provided dictionary, returning the key if the
 * provided string starts with the key or undefined otherwise. Keys are sorted by length (longest first) to ensure
 * more specific matches are checked before shorter, more general ones (e.g., "archaeology" before "arc").
 */
export const matchLoosePrefixToKey = <T extends Record<string, unknown>>(
  dict: T,
  s: string
) => {
  s = s.trim().toLowerCase();
  if (!s) {
    return undefined;
  }

  const sortedKeys = Object.keys(dict).sort((a, b) => b.length - a.length);

  const match = sortedKeys.find((rawKey) => {
    const key = rawKey.trim().toLowerCase();
    if (!key) return false;

    // Treat keys as whole words at the start: exact match or followed by space
    return s === key || s.startsWith(`${key} `);
  });

  return match ? (match as keyof T) : undefined;
};

export default {
  capitalize,
  capitalizeRole,
  pluralize,
  strToBool,
  stripQuotes,
  isIncompleteQuoteQuery,
  matchLoosePrefixToKey,
};
