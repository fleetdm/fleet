import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import DiskEncryption from "./cards/DiskEncryption";
import CustomSettings from "./cards/CustomSettings";
import { ICustomSettingsProps } from "./cards/CustomSettings/CustomSettings";
import { IDiskEncryptionProps } from "./cards/DiskEncryption/DiskEncryption";

export interface IOSSettingsCommonProps {
  currentTeamId: number;
  router: InjectedRouter; // v3
  /** handler that fires when a change occures on the section (e.g. disk encryption
   * enabled, profile uploaded) */
  onMutation: () => void;
}

type IOSSettingsCardProps = IDiskEncryptionProps | ICustomSettingsProps;

const OS_SETTINGS_NAV_ITEMS: ISideNavItem<IOSSettingsCardProps>[] = [
  {
    title: "Disk encryption",
    urlSection: "disk-encryption",
    path: PATHS.CONTROLS_DISK_ENCRYPTION,
    Card: DiskEncryption,
  },
  {
    title: "Custom settings",
    urlSection: "custom-settings",
    path: PATHS.CONTROLS_CUSTOM_SETTINGS,
    Card: CustomSettings,
  },
];

export default OS_SETTINGS_NAV_ITEMS;
