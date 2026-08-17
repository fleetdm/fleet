import { isChrome, isLinuxLike } from "interfaces/platform";
import { isEnrolledInMdm, MdmEnrollmentStatus } from "interfaces/mdm";

interface IShowControlsTabArgs {
  platform: string;
  enrollmentStatus: MdmEnrollmentStatus | null;
  /** Whether `generateTableData` derived any rows for this host. */
  hasControls: boolean;
  /** Whether the host's platform has MDM turned on globally. Omitted on the My
   * device page, whose config payload carries no per-platform MDM flags — a host
   * can't be enrolled if its platform's MDM is off, so enrollment covers it. */
  isPlatformMdmEnabled?: boolean;
}

/**
 * Whether to render the Controls tab for a host. An MDM host that is targeted by
 * no controls still gets the tab (and an empty state); an unenrolled one doesn't,
 * because the Details tab already explains why with a banner.
 */
export const shouldShowControlsTab = ({
  platform,
  enrollmentStatus,
  hasControls,
  isPlatformMdmEnabled = true,
}: IShowControlsTabArgs) => {
  if (isChrome(platform)) {
    return false;
  }

  // Linux has no MDM to enroll in, so the tab hinges on whether Fleet derived
  // any controls — in practice, disk encryption enforcement.
  if (isLinuxLike(platform)) {
    return hasControls;
  }

  return isPlatformMdmEnabled && isEnrolledInMdm(enrollmentStatus);
};

export default shouldShowControlsTab;
