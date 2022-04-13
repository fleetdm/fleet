import { bool } from "prop-types";

/**
 * Capitalizes the words of the string passed in.
 * @param str un-capitalized string
 */
const capitalize = (str: string): string => {
  return str.replace(/\b\w/g, (letter) => letter.toUpperCase());
};

/**
 * Parses the duration formated as per https://pkg.go.dev/time#ParseDuration,
 * Returns the parsed duration in milliseconds
 * @param duration duration str
 * @returns parsed duration in milliseconds
 */

const parseDuration = (duration: string): number => {
  if (duration === null || duration.length === 0) {
    throw new Error("invalid duration value");
  }

  if (duration === "0") {
    return 0;
  }

  const scales: { [unit: string]: number } = {
    ns: 1 / 1_000_000,
    us: 1 / 1_000,
    µs: 1 / 1_000,
    μs: 1 / 1_000,
    ms: 1,
    s: 1_000,
    m: 60_000,
    h: 3_600_000,
  };

  let sign = 1;
  if (duration[0] === "-") {
    sign = -1;
    duration = duration.substring(1);
  }

  let result = 0;
  let number = "";

  // eslint-disable-next-line no-restricted-syntax
  for (const c of duration) {
    if (!isNaN(parseFloat(c))) {
      number += c;
    } else if (c in scales) {
      result += parseInt(number, 10) * scales[c];
      number = "";
    } else {
      throw new Error("invalid duration value");
    }
  }

  return sign * result;
};

export default {
  capitalize,
  parseDuration,
};
