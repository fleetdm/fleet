import React, { useRef } from "react";
import classnames from "classnames";

import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";
import { MdmEnrollmentStatus } from "interfaces/mdm";

import { HostMdmDeviceStatusUIState } from "../../helpers";
import { DEVICE_STATUS_TAGS, REFETCH_TOOLTIP_MESSAGES } from "./helpers";

const baseClass = "host-header";

interface IRefetchButtonProps {
  isDisabled: boolean;
  isFetching: boolean;
  tooltip?: React.ReactNode;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
}

const RefetchButton = ({
  isDisabled,
  isFetching,
  tooltip,
  onRefetchHost,
}: IRefetchButtonProps) => {
  const classNames = classnames({
    "refetch-spinner": isFetching,
    "refetch-btn": !isFetching,
  });

  const buttonText = isFetching
    ? "Fetching fresh vitals...this may take a moment"
    : "Refetch";

  return (
    <>
      <TooltipWrapper
        underline={false}
        disableTooltip={!tooltip}
        tipContent={tooltip}
        position="top"
        showArrow
      >
        <div className={`${baseClass}__refetch`}>
          <Button
            className={classNames}
            disabled={isDisabled || isFetching}
            onClick={onRefetchHost}
            variant="inverse"
          >
            <Icon name="refresh" color="ui-fleet-black-75" size="small" />
            {buttonText}
          </Button>
        </div>
      </TooltipWrapper>
    </>
  );
};

interface IHostSummaryProps {
  summaryData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  showRefetchSpinner: boolean;
  onRefetchHost: (
    evt: React.MouseEvent<HTMLButtonElement, React.MouseEvent>
  ) => void;
  renderActionsDropdown: () => JSX.Element | null;
  deviceUser?: boolean;
  /** Optional override for the title shown when `deviceUser` is true.
   * Falls back to "My device" if not provided. */
  deviceUserHeader?: string;
  hostMdmDeviceStatus?: HostMdmDeviceStatusUIState;
  hostMdmEnrollmentStatus?: MdmEnrollmentStatus;
}

const HostHeader = ({
  summaryData,
  showRefetchSpinner,
  onRefetchHost,
  renderActionsDropdown,
  deviceUser,
  deviceUserHeader,
  hostMdmDeviceStatus,
  hostMdmEnrollmentStatus,
}: IHostSummaryProps) => {
  const { platform } = summaryData;

  const hostDisplayName = useRef<HTMLHeadingElement>(null);
  const isTruncated = useCheckTruncatedElement(hostDisplayName);

  const renderRefetch = () => {
    if (isAndroid(platform)) {
      return null;
    }

    const isOnline = summaryData.status === "online";
    let isDisabled = false;
    let tooltip;

    // we don't have a concept of "online" for iPads and iPhones
    if (!isIPadOrIPhone(platform)) {
      // deviceStatus can be `undefined` in the case of the MyDevice Page not sending
      // this prop. When this is the case or when it is `unlocked`, we only take
      // into account the host being online or offline for correctly render the
      // refresh button. If we have a value for deviceStatus, we then need to also
      // take it account for rendering the button.
      if (
        hostMdmDeviceStatus === undefined ||
        hostMdmDeviceStatus === "unlocked"
      ) {
        isDisabled = !isOnline;
        tooltip = !isOnline ? REFETCH_TOOLTIP_MESSAGES.offline : null;
      } else {
        isDisabled = true;
        tooltip = REFETCH_TOOLTIP_MESSAGES[hostMdmDeviceStatus];
      }
    } else {
      // ios and ipad devices refresh buttons disable state is determined only by the
      // host mdm device status.
      // eslint-disable-next-line
      if (
        hostMdmDeviceStatus === undefined ||
        hostMdmDeviceStatus === "unlocked" ||
        (hostMdmDeviceStatus === "locked" &&
          hostMdmEnrollmentStatus === "On (automatic)")
      ) {
        isDisabled = false;
        tooltip = null;
      } else {
        isDisabled = true;
        tooltip = REFETCH_TOOLTIP_MESSAGES[hostMdmDeviceStatus];
      }
    }

    return (
      <RefetchButton
        isDisabled={isDisabled}
        isFetching={showRefetchSpinner}
        tooltip={tooltip}
        onRefetchHost={onRefetchHost}
      />
    );
  };

  const lastFetched = summaryData.detail_updated_at ? (
    <HumanTimeDiffWithFleetLaunchCutoff
      timeString={summaryData.detail_updated_at}
    />
  ) : (
    ": unavailable"
  );

  const renderDeviceStatusTag = () => {
    if (!hostMdmDeviceStatus || hostMdmDeviceStatus === "unlocked") return null;

    // Android: per #41683 product direction, show pending/done badges (Lock pending, Wipe pending,
    // Unenroll pending, Clear passcode pending, Wiped) but NOT the "Locked" badge. AMAPI delivers
    // no "device-is-still-locked" signal: the device unlocks locally via the user's PIN with no
    // notification back to Fleet, so a "Locked" badge would be unreliable.
    if (isAndroid(platform) && hostMdmDeviceStatus === "locked") return null;

    const tag = DEVICE_STATUS_TAGS[hostMdmDeviceStatus];

    // BYO Android Unenroll fires an AMAPI WIPE under the hood (work-profile-only), so the
    // backend tracks it via wipe_ref and surfaces device_status="wiping". The admin clicked
    // Unenroll, not Wipe, so override the badge label here.
    const isAndroidBYOWipe =
      isAndroid(platform) &&
      hostMdmDeviceStatus === "wiping" &&
      hostMdmEnrollmentStatus === "On (personal)";
    const title = isAndroidBYOWipe ? "Unenroll pending" : tag.title;

    const classNames = classnames(
      `${baseClass}__device-status-tag`,
      tag.tagType
    );

    return (
      <>
        <TooltipWrapper
          tipContent={tag.generateTooltip(platform)}
          position="top"
          underline={false}
          showArrow
          className={`${baseClass}__device-status-tag-wrapper`}
        >
          <span className={classNames}>{title}</span>
        </TooltipWrapper>
      </>
    );
  };

  return (
    <div className={`${baseClass} header title`}>
      <div className="title__inner">
        <div className="display-name-container">
          <TooltipWrapper
            disableTooltip={!isTruncated}
            tipContent={
              deviceUser
                ? deviceUserHeader || "My device"
                : summaryData.display_name || DEFAULT_EMPTY_CELL_VALUE
            }
            underline={false}
            position="top"
            showArrow
          >
            <h1 className="display-name" ref={hostDisplayName}>
              {deviceUser
                ? deviceUserHeader || "My device"
                : summaryData.display_name || DEFAULT_EMPTY_CELL_VALUE}
            </h1>
          </TooltipWrapper>

          {renderDeviceStatusTag()}

          <div className={`${baseClass}__last-fetched`}>
            {"Last fetched"} {lastFetched}
            &nbsp;
          </div>
        </div>
      </div>
      <div className="title__actions">
        {renderRefetch()}
        {renderActionsDropdown()}
      </div>
    </div>
  );
};

export default HostHeader;
