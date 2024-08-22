export const SOFTWARE_PACAKGE_DROPDOWN_OPTIONS = [
  {
    label: "Download",
    value: "download",
  },
  {
    label: "Delete",
    value: "delete",
  },
  {
    label: "Advanced options",
    value: "advanced",
  },
] as const;

export const APP_STORE_APP_DROPDOWN_OPTIONS = [
  {
    label: "Delete",
    value: "delete",
  },
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
