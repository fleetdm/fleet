import React from "react";

import { IHostMdmProfile, MdmProfileStatus } from "interfaces/mdm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";

const baseClass = "os-settings-indicator";

type MdmProfileStatusForDisplay =
  | "Failed"
  | "Pending"
  | "Verifying"
  | "Verified";

interface IStatusDisplayOption {
  iconName: Extract<
    IconNames,
    "success" | "success-outline" | "pending" | "pending-outline" | "error"
  >;
}
type StatusDisplayOptions = Record<
  MdmProfileStatusForDisplay,
  IStatusDisplayOption
>;

const STATUS_DISPLAY_OPTIONS: StatusDisplayOptions = {
  Verified: {
    iconName: "success",
  },
  Verifying: {
    iconName: "success-outline",
  },
  Pending: {
    iconName: "pending-outline",
  },
  Failed: {
    iconName: "error",
  },
};

const countHostProfilesByStatus = (
  hostSettings: IHostMdmProfile[]
): Record<MdmProfileStatus, number> => {
  return hostSettings.reduce(
    (acc, { status }) => {
      if (status === "failed") {
        acc.failed += 1;
      } else if (status === "pending" || status === "action_required") {
        acc.pending += 1;
      } else if (status === "verifying") {
        acc.verifying += 1;
      } else if (status === "verified") {
        acc.verified += 1;
      }

      return acc;
    },
    {
      failed: 0,
      pending: 0,
      verifying: 0,
      verified: 0,
    }
  );
};

/**
 * Returns the displayed status of the macOS settings field based on the
 * profile statuses.
 * If any profile has a status of "failed", the status will be displayed as "Failed" and
 * continues to fall through to "Pending" and "Verifying" if any profiles have those statuses.
 * If all profiles have a status of "verified", the status will be displayed as "Verified".
 *
 * The default status will be displayed as "Failed".
 * https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks
 */
const getHostProfilesStatusForDisplay = (
  hostMacSettings: IHostMdmProfile[]
): MdmProfileStatusForDisplay => {
  const counts = countHostProfilesByStatus(hostMacSettings);
  switch (true) {
    case !!counts.failed:
      return "Failed";
    case !!counts.pending:
      return "Pending";
    case !!counts.verifying:
      return "Verifying";
    case counts.verified === hostMacSettings.length:
      return "Verified";
    default:
      // something is broken
      return "Failed";
  }
};

interface IOSSettingsIndicatorProps {
  profiles: IHostMdmProfile[];
  onClick?: () => void;
}
const OSSettingsIndicator = ({
  profiles,
  onClick,
}: IOSSettingsIndicatorProps): JSX.Element => {
  if (!profiles.length) {
    // the caller should ensure that this never happens, but just in case we return a default
    // to make it more obvious that something is wrong.
    // https://fleetdm.com/handbook/company/why-this-way#why-make-it-obvious-when-stuff-breaks
    return <span className={`${baseClass} info-flex__data`}>Unavailable</span>;
  }

  const displayStatus = getHostProfilesStatusForDisplay(profiles);

  const statusDisplayOption = STATUS_DISPLAY_OPTIONS[displayStatus];

  return (
    <span className={`${baseClass} info-flex__data`}>
      <Icon name={statusDisplayOption.iconName} />
      <Button
        onClick={onClick}
        variant="text-link"
        className={`${baseClass}__button`}
      >
        {displayStatus}
      </Button>
    </span>
  );
};

export default OSSettingsIndicator;
