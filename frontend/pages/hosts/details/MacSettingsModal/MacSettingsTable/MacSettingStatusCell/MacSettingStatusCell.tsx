import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IconNames } from "components/icons";
import { MacMdmProfileOperationType, MdmProfileStatus } from "interfaces/mdm";
import _ from "lodash";
import React from "react";
import ReactTooltip from "react-tooltip";

const baseClass = "mac-setting-status-cell";

type ProfileDisplayOption = {
  statusText: string;
  iconName: IconNames;
  tooltipText?: (isDiskEncryption: boolean) => string;
} | null;

type OperationTypeOption = Record<MdmProfileStatus, ProfileDisplayOption>;
type ProfileDisplayConfig = Record<
  MacMdmProfileOperationType,
  OperationTypeOption
>;

const PROFILE_DISPLAY_CONFIG: ProfileDisplayConfig = {
  install: {
    verified: {
      statusText: "Verified",
      iconName: "success",
      tooltipText: (isDiskEncryption) =>
        isDiskEncryption
          ? "The host turned disk encryption on and " +
            "sent their key to Fleet. Fleet verified with osquery."
          : "The host installed the configuration profile. Fleet verified with osquery.",
    },
    verifying: {
      statusText: "Verifying",
      iconName: "success-partial",
      tooltipText: (isDiskEncryption) =>
        isDiskEncryption
          ? "The host acknowledged the MDM command to install disk encryption profile. Fleet is " +
            "verifying with osquery and retrieving the disk encryption key. This may take up to one hour."
          : "The host acknowledged the MDM command to install the configuration profile. Fleet is " +
            "verifying with osquery.",
    },
    pending: {
      statusText: "Enforcing (pending)",
      iconName: "pending-partial",
      tooltipText: (isDiskEncryption) =>
        isDiskEncryption
          ? "The host will receive the MDM command to install the disk encryption profile when the " +
            "host comes online."
          : "The host will receive the MDM command to install the configuration profile when the " +
            "host comes online.",
    },
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltipText: undefined,
    },
  },
  remove: {
    pending: {
      statusText: "Removing enforcement (pending)",
      iconName: "pending-partial",
      tooltipText: (isDiskEncryption) =>
        isDiskEncryption
          ? "The host will receive the MDM command to remove the disk encryption profile when the " +
            "host comes online."
          : "The host will receive the MDM command to remove the configuration profile when the host " +
            "comes online.",
    },
    verified: null, // should not be reached
    verifying: null, // should not be reached
    failed: {
      statusText: "Failed",
      iconName: "error",
      tooltipText: undefined,
    },
  },
};

interface IMacSettingStatusCellProps {
  name: string;
  status: MdmProfileStatus;
  operationType: MacMdmProfileOperationType;
}
const MacSettingStatusCell = ({
  name,
  status,
  operationType,
}: IMacSettingStatusCellProps): JSX.Element => {
  const options = PROFILE_DISPLAY_CONFIG[operationType]?.[status];

  const isDiskEncryptionProfile = name === "Disk Encryption";

  if (options) {
    const { statusText, iconName, tooltipText } = options;
    const tooltipId = _.uniqueId();
    return (
      <span className={baseClass}>
        <Icon name={iconName} />
        {tooltipText ? (
          <>
            <span
              className="tooltip tooltip__tooltip-icon"
              data-tip
              data-for={tooltipId}
              data-tip-disable={false}
            >
              {statusText}
            </span>
            <ReactTooltip
              place="top"
              effect="solid"
              backgroundColor="#3e4771"
              id={tooltipId}
              data-html
            >
              <span className="tooltip__tooltip-text">
                {tooltipText(isDiskEncryptionProfile)}
              </span>
            </ReactTooltip>
          </>
        ) : (
          statusText
        )}
      </span>
    );
  }
  // graceful error - this state should not be reached based on the API spec
  return <TextCell value="Unrecognized" />;
};
export default MacSettingStatusCell;
