import linuxIcon from "../../../../../../assets/images/icon-linux-fleet-black-16x16@2x.png";
import darwinIcon from "../../../../../../assets/images/icon-darwin-fleet-black-16x16@2x.png";
import windowsIcon from "../../../../../../assets/images/icon-windows-fleet-black-16x16@2x.png";

export const EMPTY_OPTION = {
  label: "No Matching Labels",
  isDisabled: true,
};

export const PLATFORM_TYPE_ICONS: Record<string, any> = {
  "All Linux": linuxIcon,
  macOS: darwinIcon,
  "MS Windows": windowsIcon,
};

export const FILTERED_LINUX = ["Red Hat Linux", "CentOS Linux", "Ubuntu Linux"];
