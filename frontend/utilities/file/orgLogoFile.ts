export const ORG_LOGO_ACCEPT = ".png,.jpg,.jpeg,.webp";
export const ORG_LOGO_MAX_SIZE_BYTES = 100 * 1024; // 100 KB
export const ORG_LOGO_HELP_TEXT =
  "PNG, JPEG, or WebP file. For best results, use a square PNG at least 150x150 px (transparency).";

export interface IOrgLogoValidationResult {
  valid: boolean;
  error?: string;
}

const PNG_MAGIC = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];

// hasAcceptedImageMagic checks whether the leading bytes match one of the
// formats supported by the BE (PNG, JPEG, or WebP). The browser-reported
// `file.type` is based on extension, not content, so we sniff the bytes
// here to catch e.g. a WebP saved with a `.png` filename.
const hasAcceptedImageMagic = (bytes: Uint8Array): boolean => {
  // PNG: 89 50 4E 47 0D 0A 1A 0A
  if (bytes.length >= 8 && PNG_MAGIC.every((b, i) => bytes[i] === b)) {
    return true;
  }
  // JPEG: FF D8 FF
  if (
    bytes.length >= 3 &&
    bytes[0] === 0xff &&
    bytes[1] === 0xd8 &&
    bytes[2] === 0xff
  ) {
    return true;
  }
  // WebP: "RIFF" at 0..3, "WEBP" at 8..11 (4..7 carries the file size).
  if (
    bytes.length >= 12 &&
    bytes[0] === 0x52 &&
    bytes[1] === 0x49 &&
    bytes[2] === 0x46 &&
    bytes[3] === 0x46 &&
    bytes[8] === 0x57 &&
    bytes[9] === 0x45 &&
    bytes[10] === 0x42 &&
    bytes[11] === 0x50
  ) {
    return true;
  }
  return false;
};

export const validateOrgLogoFile = async (
  file: File
): Promise<IOrgLogoValidationResult> => {
  if (file.size > ORG_LOGO_MAX_SIZE_BYTES) {
    return { valid: false, error: "Logo must be 100 KB or less." };
  }
  // Read the first 12 bytes — enough to cover PNG (8), JPEG (3), and WebP
  // (which needs offset 8..11 for the "WEBP" marker).
  const headerBuf = await file.slice(0, 12).arrayBuffer();
  const header = new Uint8Array(headerBuf);
  if (!hasAcceptedImageMagic(header)) {
    return {
      valid: false,
      error: "Logo must be a PNG, JPEG, or WebP file.",
    };
  }
  return { valid: true };
};
