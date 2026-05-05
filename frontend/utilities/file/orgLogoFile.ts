export const ORG_LOGO_ACCEPT = ".png,.jpg,.jpeg,.webp,.svg";
export const ORG_LOGO_MAX_SIZE_BYTES = 100 * 1024; // 100 KB
export const ORG_LOGO_HELP_TEXT =
  "Personalize Fleet with your brand. For best results, use a square image at least 150px wide.";
export const ORG_LOGO_ALLOWED_TYPES = ["png", "jpeg", "webp", "svg"] as const;

export type ImageFileType = typeof ORG_LOGO_ALLOWED_TYPES[number];

const upperAllowedTypes = ORG_LOGO_ALLOWED_TYPES.map((t) => t.toUpperCase());
const ORG_LOGO_ALLOWED_TYPES_LABEL = `${upperAllowedTypes
  .slice(0, -1)
  .join(", ")}, or ${upperAllowedTypes[upperAllowedTypes.length - 1]}`;

// Larger than any non-SVG magic-byte prefix so a single read covers all
// formats. SVG detection scans the head for "<svg" anywhere, allowing
// for an XML declaration / comments / DOCTYPE before the root tag.
const SNIFF_BYTES = 1024;

export interface IOrgLogoValidationResult {
  valid: boolean;
  error?: string;
}

const PNG_MAGIC = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];

const detectImageType = (bytes: Uint8Array): ImageFileType | null => {
  // PNG: 89 50 4E 47 0D 0A 1A 0A
  if (bytes.length >= 8 && PNG_MAGIC.every((b, i) => bytes[i] === b)) {
    return "png";
  }
  // JPEG: FF D8 FF
  if (
    bytes.length >= 3 &&
    bytes[0] === 0xff &&
    bytes[1] === 0xd8 &&
    bytes[2] === 0xff
  ) {
    return "jpeg";
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
    return "webp";
  }
  // SVG is text — search the sniff window for "<svg" (case-insensitive).
  // Real SVGs put the root tag near the top, after at most an XML
  // declaration, comments, or a DOCTYPE. Strict safety checks happen
  // server-side; the FE check is just for early UX feedback.
  const text = new TextDecoder("utf-8", { fatal: false }).decode(bytes);
  if (/<svg\b/i.test(text)) {
    return "svg";
  }
  return null;
};

// validateOrgLogoFile sniffs the leading bytes of a File to verify
// it's one of the allowed image formats — the browser-reported
// file.type is based on extension, not content, so we can't trust it
// (e.g. a WebP saved with a `.png` extension).
export const validateOrgLogoFile = async (
  file: File
): Promise<IOrgLogoValidationResult> => {
  if (file.size > ORG_LOGO_MAX_SIZE_BYTES) {
    return { valid: false, error: "Logo must be 100 KB or less." };
  }
  const headerBuf = await file.slice(0, SNIFF_BYTES).arrayBuffer();
  const detected = detectImageType(new Uint8Array(headerBuf));
  if (!detected || !ORG_LOGO_ALLOWED_TYPES.includes(detected)) {
    return {
      valid: false,
      error: `Logo must be a ${ORG_LOGO_ALLOWED_TYPES_LABEL} file.`,
    };
  }
  return { valid: true };
};
