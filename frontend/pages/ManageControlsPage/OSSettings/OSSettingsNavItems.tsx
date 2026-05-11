import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import DiskEncryption from "./cards/DiskEncryption";
import ConfigurationProfiles from "./cards/ConfigurationProfiles";
import Certificates from "./cards/Certificates";
import Passwords from "./cards/Passwords";
import { IConfigurationProfilesProps } from "./cards/ConfigurationProfiles/ConfigurationProfiles";
import { IDiskEncryptionProps } from "./cards/DiskEncryption/DiskEncryption";

export interface IOSSettingsCommonProps {
  currentTeamId: number;
  router: InjectedRouter;
  /** handler that fires when a change occures on the section (e.g. disk encryption
   * enabled, profile uploaded) */
  onMutation: () => void;
}

type IOSSettingsCardProps = IDiskEncryptionProps | IConfigurationProfilesProps;

// Observers and observers+ will not have access to the Controls page at all, so the only role to
// exclude at this point is technician
const getOSSettingsNavItems = (
  isTechnician: boolean
): ISideNavItem<IOSSettingsCardProps>[] => {
  const items = [
    {
      title: "Disk encryption",
      urlSection: "disk-encryption",
      path: PATHS.CONTROLS_DISK_ENCRYPTION,
      Card: DiskEncryption,
    },
    {
      title: "Configuration profiles",
      urlSection: "configuration-profiles",
      path: PATHS.CONTROLS_CUSTOM_SETTINGS,
      Card: ConfigurationProfiles,
    },
    {
      title: "Certificates",
      Card: Certificates,
      urlSection: "certificates",
      path: PATHS.CONTROLS_CERTIFICATES,
      exclude: isTechnician,
    },
    {
      title: "Passwords",
      Card: Passwords,
      urlSection: "passwords",
      path: PATHS.CONTROLS_PASSWORDS,
      exclude: isTechnician,
    },
  ];
  return items.filter((item) => !item.exclude);
};

export default getOSSettingsNavItems;
