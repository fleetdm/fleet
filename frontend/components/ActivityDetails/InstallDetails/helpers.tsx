/** Safely decode a base64-encoded UTF-8 string into a JavaScript string.
 * atob returns a binary string (bytes 0–255), so wrap it in Uint8Array and
 * use TextDecoder to handle multi-byte UTF-8 characters. */
const decodeBase64Utf8 = (b64: string) => {
  const bin = atob(b64); // binary string (bytes 0–255)
  const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
  return new TextDecoder("utf-8").decode(bytes);
};
export default decodeBase64Utf8;
