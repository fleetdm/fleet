// Header name for signaling base64-encoded scripts to bypass WAF rules
// that may block requests containing shell/PowerShell script patterns.
export const SCRIPTS_ENCODED_HEADER = "X-Fleet-Scripts-Encoded";

/**
 * Base64 encode a script string with proper UTF-8 handling.
 * Returns undefined for undefined input and empty string for empty input,
 * which allows callers to pass through empty/unset script fields without modification.
 */
export const encodeScriptBase64 = (
  script: string | undefined
): string | undefined => {
  if (script === undefined) {
    return undefined;
  }
  if (script === "") {
    return "";
  }
  // Use TextEncoder for proper UTF-8 handling of unicode characters
  const encoder = new TextEncoder();
  const data = encoder.encode(script);
  // Convert Uint8Array to binary string, then base64
  let binary = "";
  data.forEach((byte) => {
    binary += String.fromCharCode(byte);
  });
  return btoa(binary);
};
