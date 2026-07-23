import {
  FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME,
  getExtensionFromFileName,
} from "utilities/file/fileUtils";

/** What the file-uploader should accept and how the file-type hint reads when
 * adding a new package to a multi-package title. The new upload must match the
 * existing title's platform/file type, so we derive both from the first-added
 * package's filename. Returns `null` when we can't determine the restriction
 * (unknown extension or missing name) so callers fall back to PackageForm's
 * full all-platforms accept + message. */
export interface IFileTypeRestriction {
  /** Value for `<input type="file" accept>` — narrowed to a single extension
   * (or, for tar.gz, the same MIME/extension pair PackageForm uses globally). */
  accept: string;
  /** Display label, e.g. `"macOS (.pkg)"` or `"Linux (.tar.gz)"`. */
  label: string;
}

export const getFileTypeRestriction = (
  existingPackageName: string
): IFileTypeRestriction | null => {
  const extension = getExtensionFromFileName(existingPackageName);
  if (!extension) return null;

  const platform = FILE_EXTENSIONS_TO_PLATFORM_DISPLAY_NAME[extension];
  if (!platform) return null;

  // Browsers can't reliably match `.tar.gz` via extension alone (double-
  // extension). Mirror PackageForm's global accept value for tar.gz so the
  // file dialog filters correctly without us reimplementing the workaround.
  const accept =
    extension === "tar.gz" ? "application/gzip,.tgz" : `.${extension}`;

  return {
    accept,
    label: `${platform} (.${extension})`,
  };
};
