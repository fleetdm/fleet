const REGIONAL_INDICATOR_OFFSET = 0x1f1e6 - "A".charCodeAt(0);

/** Returns the flag emoji for an ISO 3166-1 alpha-2 country code, e.g. "jp" -> "🇯🇵". */
export const getCountryFlag = (countryCode: string) =>
  String.fromCodePoint(
    ...[...countryCode.toUpperCase()].map(
      (char) => char.charCodeAt(0) + REGIONAL_INDICATOR_OFFSET
    )
  );

const getCountryName = (countryCode: string) => {
  try {
    return new Intl.DisplayNames(["en"], { type: "region" }).of(
      countryCode.toUpperCase()
    );
  } catch {
    return undefined;
  }
};

/**
 * Returns the flag and English name for an ISO 3166-1 alpha-2 country code,
 * e.g. "jp" -> "🇯🇵 Japan". Falls back to the uppercase code when the name is
 * unknown, and returns undefined for an empty code.
 */
export const getCountryDisplay = (countryCode?: string) => {
  if (!countryCode) {
    return undefined;
  }
  const code = countryCode.toUpperCase();
  if (!/^[A-Z]{2}$/.test(code)) {
    return code;
  }
  const name = getCountryName(code);
  if (!name || name === code) {
    return code;
  }
  return `${getCountryFlag(code)} ${name}`;
};
