const REGION_NAMES = new Intl.DisplayNames(["en"], { type: "region" });

/** "jp" -> "🇯🇵 Japan". Falls back to the uppercased code. */
export const formatCountry = (code?: string | null): string => {
  if (!code) return "";
  const upper = code.toUpperCase();
  if (!/^[A-Z]{2}$/.test(upper)) return upper;
  let name = upper;
  try {
    name = REGION_NAMES.of(upper) ?? upper;
  } catch {
    // Invalid region subtag. Keep the code.
  }
  // Regional indicator symbols: "A" (65) maps to U+1F1E6.
  const flag = String.fromCodePoint(
    ...[...upper].map((c) => c.charCodeAt(0) + 0x1f1a5)
  );
  return `${flag} ${name}`;
};

export default formatCountry;
