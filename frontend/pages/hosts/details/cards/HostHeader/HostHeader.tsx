import React, { useRef } from "react";
import classnames from "classnames";

import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";

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
  hostMdmDeviceStatus?: HostMdmDeviceStatusUIState;
}

const HostHeader = ({
  summaryData,
  showRefetchSpinner,
  onRefetchHost,
  renderActionsDropdown,
  deviceUser,
  hostMdmDeviceStatus,
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
        hostMdmDeviceStatus === "unlocked"
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

    const tag = DEVICE_STATUS_TAGS[hostMdmDeviceStatus];

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
        >
          <span className={classNames}>{tag.title}</span>
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
                ? "My device"
                : summaryData.display_name || DEFAULT_EMPTY_CELL_VALUE
            }
            underline={false}
            position="top"
            showArrow
          >
            <h1 className="display-name" ref={hostDisplayName}>
              {deviceUser
                ? "My device"
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
