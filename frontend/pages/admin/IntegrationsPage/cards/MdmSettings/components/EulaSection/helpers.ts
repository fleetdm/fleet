import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { Platform } from "interfaces/platform";
import mdmAPI from "services/entities/mdm";

export type EulaPlatform = Extract<Platform, "darwin" | "windows">;

export const EULA_PLATFORMS: EulaPlatform[] = ["darwin", "windows"];

interface IEulaPlatformConfig {
  extension: ".pdf" | ".md";
  /** Checked before upload; the server enforces the same limit. */
  sizeLimit?: { bytes: number; message: string };
  graphicName: "file-pdf" | "file-md";
  uploadMessage: string;
  wrongTypeMessage: string;
  unavailableMessage: string;
  deleteMessage: string;
  /**
   * The macOS PDF opens in the browser. The Windows document is markdown that
   * the enrollment page renders, so an example of that page replaces the
   * preview.
   */
  hasExample: boolean;
  upload: (file: File) => Promise<unknown>;
  remove: (token: string) => Promise<unknown>;
}

export const EULA_PLATFORM_CONFIG: Record<EulaPlatform, IEulaPlatformConfig> = {
  darwin: {
    extension: ".pdf",
    graphicName: "file-pdf",
    uploadMessage: "PDF (.pdf)",
    wrongTypeMessage: "Couldn’t upload EULA. The file must be a PDF (.pdf).",
    unavailableMessage:
      "To require a EULA on macOS hosts, first turn on Apple (macOS, iOS, iPadOS) MDM and add an Apple Business Manager token.",
    deleteMessage:
      "End users won’t be required to agree to this EULA on macOS hosts that automatically enroll.",
    hasExample: false,
    upload: (file) => mdmAPI.uploadEULA(file),
    remove: (token) => mdmAPI.deleteEULA(token),
  },
  windows: {
    extension: ".md",
    sizeLimit: {
      bytes: 512 * 1024,
      message: "Couldn’t upload EULA. The file must be 512 KB or smaller.",
    },
    graphicName: "file-md",
    uploadMessage:
      "Export your document as a markdown (.md) file and upload it.",
    wrongTypeMessage:
      "Couldn’t upload EULA. The file must be a markdown (.md) file.",
    unavailableMessage:
      "To require terms on Windows hosts, first turn on Windows MDM.",
    deleteMessage:
      "End users won’t be required to agree to these terms on Windows hosts that enroll through Microsoft Entra. Fleet’s default terms will be shown instead.",
    hasExample: true,
    upload: (file) => mdmAPI.uploadWindowsEULA(file),
    remove: (token) => mdmAPI.deleteWindowsEULA(token),
  },
};

const DEFAULT_ERROR_MESSAGE = "Couldn’t upload EULA. Please try again.";

// Server reasons that are already written for end users and are shown as is.
const PASSTHROUGH_REASONS = [
  "The file is empty.",
  "The file must be UTF-8 text.",
  "The file contains HTML. Convert it to markdown and upload again.",
  "The file has no text to show.",
  "The file must be 512 KB or smaller.",
  "The file has a line longer than 8 KB. Split long paragraphs into shorter lines and upload again.",
  "The file has more than 10,000 lines.",
  "The file has too much formatting in one paragraph or list. Add blank lines between paragraphs and upload again.",
  "The file nests lists or quotes too deeply.",
  "The file's tables have more than 10,000 cells in total.",
  "The file is too large to show. Make it shorter and upload again.",
  "The file couldn't be read as markdown.",
  "The file name must be 255 characters or fewer.",
];

export const hasExpectedExtension = (
  platform: EulaPlatform,
  fileName: string
) => fileName.toLowerCase().endsWith(EULA_PLATFORM_CONFIG[platform].extension);

export const getErrorMessage = (
  platform: EulaPlatform,
  err: AxiosResponse<IApiError>
) => {
  const reason = err.data?.errors?.[0]?.reason ?? "";

  if (
    reason.includes("invalid file type") ||
    reason.includes("must be a markdown")
  ) {
    return EULA_PLATFORM_CONFIG[platform].wrongTypeMessage;
  }
  if (PASSTHROUGH_REASONS.some((r) => reason.includes(r))) {
    return `Couldn’t upload EULA. ${reason}`;
  }
  return DEFAULT_ERROR_MESSAGE;
};
