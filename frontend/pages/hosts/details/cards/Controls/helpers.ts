import { isEnrolledInMdm, MdmEnrollmentStatus } from "interfaces/mdm";
import {
  HostPlatform,
  isChrome,
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
   * config has no per-platform flags — enrollment implies it. */
  isPlatformMdmEnabled?: boolean;
}

/**
 * Whether to render the Controls tab for a host: whenever Fleet derived any
 * rows, plus on an enrolled host with none, where the tab earns its empty state.
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
  // before this tab existed. Also covers Linux, which has no MDM to enroll in.
  if (hasControls) {
    return true;
  }

  // With nothing to show, only a host that could receive controls earns the
  // empty state. An unenrolled one gets the Details tab's yellow banner.
  return isPlatformMdmEnabled && isEnrolledInMdm(enrollmentStatus);
};

export default shouldShowControlsTab;
