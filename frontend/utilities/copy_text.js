import select from "select";

const removeSelectedText = () => {
  return global.window.getSelection().removeAllRanges();
};

export const copyText = (elementSelector) => {
  const { document } = global;

  const element = document.querySelector(elementSelector);
  const input = element.querySelector("input");
  input.type = "text";
  input.disabled = false;

  select(input);

  const canCopy = document.queryCommandEnabled("copy");

  if (!canCopy) {
    return false;
  }

  document.execCommand("copy");
  input.type = "password";
  input.disabled = true;
  removeSelectedText();
  return true;
};

export const stringToClipboard = (string) => {
  const { navigator } = global;

  return navigator.clipboard.writeText(string);
};

export const COPY_TEXT_SUCCESS = "Text copied to clipboard";
export const COPY_TEXT_ERROR = "Text not copied. Please copy manually.";

export default copyText;
