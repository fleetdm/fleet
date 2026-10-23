const PLIST_START = /<\?xml|<!DOCTYPE plist|<plist[\s>]/;
const PLIST_END = "</plist>";
// eslint-disable-next-line no-control-regex
const CONTROL_CHARS = /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/; // Matches ASCII control characters except tab, \n, and \r

const decodeBase64Bytes = (b64: string): Uint8Array | null => {
  try {
    const bin = atob(b64.replace(/\s/g, ""));
    return Uint8Array.from(bin, (c) => c.charCodeAt(0));
  } catch {
    return null;
  }
};

/** Returns the readable text in a plist `<data>` value, or null if it's
 * binary. Fleet signs profiles before sending them, so an InstallProfile
 * payload is a PKCS #7 envelope with the profile XML stored verbatim inside. */
const decodeDataValue = (b64: string): string | null => {
  const bytes = decodeBase64Bytes(b64);
  if (!bytes?.length) {
    return null;
  }

  try {
    const text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    if (!CONTROL_CHARS.test(text)) {
      return text;
    }
  } catch {
    // not UTF-8 text; look for a plist embedded in the binary below
  }

  // a lossy decode leaves the ASCII markers around the embedded plist intact
  const lossy = new TextDecoder("utf-8").decode(bytes);
  const start = lossy.search(PLIST_START);
  const end = lossy.lastIndexOf(PLIST_END);
  if (start === -1 || end < start) {
    return null;
  }
  return lossy.slice(start, end + PLIST_END.length);
};

/**
 * Decodes the base64 `<data>` values in an Apple MDM command so it can be read
 * as sent, e.g. the profile in an InstallProfile command. Values that aren't
 * text, like certificates, are left as base64.
 */
const decodePlistDataForDisplay = (plist: string): string =>
  plist.replace(
    /<data>([^<]*)<\/data>/g,
    (match, b64: string, offset: number) => {
      const decoded = decodeDataValue(b64)?.trim();
      if (!decoded) {
        return match;
      }
      if (!decoded.includes("\n")) {
        return `<data>${decoded}</data>`;
      }

      const lineStart = plist.lastIndexOf("\n", offset) + 1;
      const indent = /^[ \t]*/.exec(plist.slice(lineStart, offset))?.[0] ?? "";
      const body = decoded
        .split(/\r?\n/)
        .map((line) => (line ? `${indent}\t${line}` : line))
        .join("\n");
      return `<data>\n${body}\n${indent}</data>`;
    }
  );

export default decodePlistDataForDisplay;
