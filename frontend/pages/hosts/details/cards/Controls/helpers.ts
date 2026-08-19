import { isEnrolledInMdm, MdmEnrollmentStatus } from "interfaces/mdm";
import {
  HostPlatform,
  isChrome,
  isLinuxLike,
  isOsSettingsDisplayPlatform,
} from "interfaces/platform";

interface IShowControlsTabArgs {
  platform: HostPlatform;
  /** Needed to tell Fedora from other rhel-like platforms. */
  osVersion: string;
  enrollmentStatus: MdmEnrollmentStatus | null;
  /** Whether `generateTableData` derived any rows for this host. */
  hasControls: boolean;
  /** Whether the platform's MDM is on globally. Omitted on My device, whose
   * config carries no per-platform flags; treated as on there. */
  isPlatformMdmEnabled?: boolean;
}

/**
 * Whether to render the Controls tab for a host: whenever Fleet derived any
 * rows, plus wherever the platform can receive controls at all, where the tab
 * earns its empty state.
 */
export const shouldShowControlsTab = ({
  platform,
  osVersion,
  enrollmentStatus,
  hasControls,
  isPlatformMdmEnabled = true,
}: IShowControlsTabArgs) => {
  // ChromeOS is in the OS-settings platform list but the tab is hidden for it.
  if (isChrome(platform) || !isOsSettingsDisplayPlatform(platform, osVersion)) {
    return false;
  }

  // Any derived row shows the tab — the bar the OS settings indicator used
  // before this tab existed.
  if (hasControls) {
    return true;
  }

  // Linux has no MDM, so there's no enrollment state to explain and nothing
  // else it can ever be targeted by. Checked before the MDM branch below, whose
  // `isPlatformMdmEnabled` defaults to true on My device.
  if (isLinuxLike(platform)) {
    return false;
  }

  // With nothing to show, the tab still earns its empty state wherever the
  // platform's MDM is on — hiding it would leave an unenrolled host with no
  // explanation, since the "Turn on MDM" banner is macOS-only. The enrollment
  // check covers a host still reading as enrolled after MDM was turned off.
  return isPlatformMdmEnabled || isEnrolledInMdm(enrollmentStatus);
};

export default shouldShowControlsTab;
