import { IconNames } from "components/icons";
import { ReactNode } from "react";

type ISoftwareOption = {
  value: string;
  disabled: boolean;
  iconName: IconNames;
  tooltipContent?: ReactNode;
};

const DOWNLOAD_OPTION: ISoftwareOption = {
  value: "download",
  disabled: false,
  iconName: "download",
};

const EDIT_OPTION: ISoftwareOption = {
  value: "edit",
  disabled: false,
  iconName: "pencil",
};
const DELETE_OPTION: ISoftwareOption = {
  value: "delete",
  disabled: false,
  iconName: "trash",
};

export const SOFTWARE_PACKAGE_ACTION_OPTIONS = [
  DOWNLOAD_OPTION,
  EDIT_OPTION,
  DELETE_OPTION,
] as const;

export const APP_STORE_APP_ACTION_OPTIONS = [
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
