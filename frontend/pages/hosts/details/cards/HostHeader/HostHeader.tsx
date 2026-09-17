import React, { useRef } from "react";
import classnames from "classnames";

import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import Button from "components/buttons/Button";
import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { useCheckTruncatedElement } from "hooks/useCheckTruncatedElement";
import TooltipWrapper from "components/TooltipWrapper";
import { MdmEnrollmentStatus } from "interfaces/mdm";
import { humanLastSeen, internationalTimeFormat } from "utilities/helpers";

import { HostMdmDeviceStatusUIState } from "../../helpers";
import {
  ANDROID_NO_REFETCH_TOOLTIP_MESSAGE,
  DEVICE_STATUS_TAGS,
  REFETCH_TOOLTIP_MESSAGES,
} from "./helpers";

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
            variant="secondary"
            icon="refresh"
          >
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
  hostMdmEnrollmentStatus: MdmEnrollmentStatus | null;
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
      return (
        <RefetchButton
          isDisabled
          isFetching={false}
          tooltip={ANDROID_NO_REFETCH_TOOLTIP_MESSAGE}
          onRefetchHost={onRefetchHost}
        />
      );
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

  // `summaryData` is run through `normalizeEmptyValues`, so a host that has
  // never checked in reports "---", while platforms whose API response omits
  // the field altogether leave it undefined.
  const lastMdmCheckIn = summaryData.last_mdm_checked_in_at;
  const hasMdmCheckIn =
    !!lastMdmCheckIn && lastMdmCheckIn !== DEFAULT_EMPTY_CELL_VALUE;

  // "Last fetched" is the most recent of the osquery detail refresh and the
  // policy refresh, so a patch-when-closed skip driven by a fresh policy
  // re-eval doesn't look older than the page itself.
  const detailUpdatedAtMs = Date.parse(summaryData.detail_updated_at);
  const policyUpdatedAtMs = Date.parse(summaryData.policy_updated_at);
  const detailValid = Number.isFinite(detailUpdatedAtMs);
  const policyValid = Number.isFinite(policyUpdatedAtMs);
  let lastFetchedAt: string | undefined;
  if (detailValid && policyValid) {
    lastFetchedAt = new Date(
      Math.max(detailUpdatedAtMs, policyUpdatedAtMs)
    ).toISOString();
  } else if (detailValid) {
    lastFetchedAt = summaryData.detail_updated_at;
  } else if (policyValid) {
    lastFetchedAt = summaryData.policy_updated_at;
  }

  const withTooltip = (content: React.ReactNode) => {
    if (!hasMdmCheckIn) {
      return content;
    }

    return (
      <TooltipWrapper
        tipContent={
          <>
            <span>
              Last fetched:
              <b>
                {" "}
                {lastFetchedAt
                  ? internationalTimeFormat(new Date(lastFetchedAt))
                  : "unavailable"}
              </b>
            </span>
            <br />
            <span>
              Last MDM check-in:
              <b> {internationalTimeFormat(new Date(lastMdmCheckIn))}</b>
            </span>
          </>
        }
        position="top"
        underline={false}
        showArrow
      >
        {content}
      </TooltipWrapper>
    );
  };

  let lastFetched: React.ReactNode = ": unavailable";
  if (lastFetchedAt && hasMdmCheckIn) {
    lastFetched = humanLastSeen(lastFetchedAt);
  } else if (lastFetchedAt) {
    lastFetched = (
      <HumanTimeDiffWithFleetLaunchCutoff timeString={lastFetchedAt} />
    );
  }

  const renderDeviceStatusTag = () => {
    if (!hostMdmDeviceStatus || hostMdmDeviceStatus === "unlocked") return null;
    const tag = DEVICE_STATUS_TAGS[hostMdmDeviceStatus];

    const title = tag.title;
    const tipContent = tag.generateTooltip(platform);

    const classNames = classnames(
      `${baseClass}__device-status-tag`,
      tag.tagType
    );

    return (
      <>
        <TooltipWrapper
          tipContent={tipContent}
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
            {withTooltip(
              <>
                {"Last fetched"} {lastFetched}
              </>
            )}
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
