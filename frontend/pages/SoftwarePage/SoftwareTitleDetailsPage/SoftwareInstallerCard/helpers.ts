const DOWNLOAD_OPTION = {
  label: "Download",
  value: "download",
};

const EDIT_OPTION = {
  label: "Edit",
  value: "edit",
};
const DELETE_OPTION = {
  label: "Delete",
  value: "delete",
};

export const SOFTWARE_PACKAGE_DROPDOWN_OPTIONS = [
  DOWNLOAD_OPTION,
  EDIT_OPTION,
  DELETE_OPTION,
] as const;

export const APP_STORE_APP_DROPDOWN_OPTIONS = [
  EDIT_OPTION,
  DELETE_OPTION,
] as const;

export const downloadFile = (url: string, fileName: string) => {
  // Download a file by simulating a link click.
  const downloadLink = document.createElement("a");
  downloadLink.href = url;
  downloadLink.download = fileName;
  downloadLink.click();

  // Clean up above-created "a" element
  downloadLink.remove();
};
