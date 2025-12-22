const decodeBase64Utf8 = (b64: string) => {
  const bin = atob(b64); // binary string (bytes 0–255)
  const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
  return new TextDecoder("utf-8").decode(bytes);
};
export default decodeBase64Utf8;
