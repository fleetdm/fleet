// @ts-nocheck - may need to be reworked
export const stringToClipboard = (string) => {
  const { navigator } = global;

  return navigator.clipboard.writeText(string);
};

export const COPY_TEXT_SUCCESS = "Text copied to clipboard";
export const COPY_TEXT_ERROR = "Text not copied. Please copy manually.";
