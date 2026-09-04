import React from "react";
import classnames from "classnames";
import { formatInTimeZone } from "date-fns-tz";
import { BootstrapPackageStatus } from "interfaces/mdm";
import { IHostMaintenanceWindow } from "interfaces/host";
import { isAndroid, isIPadOrIPhone } from "interfaces/platform";

import { getHostStatus, getHostStatusTooltipText } from "pages/hosts/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Card from "components/Card";
import DataSet from "components/DataSet";
import StatusIndicator from "components/StatusIndicator";
import Button from "components/buttons/Button";
import IssuesIndicator from "pages/hosts/components/IssuesIndicator";

import {
  DATE_FNS_FORMAT_STRINGS,
  DEFAULT_EMPTY_CELL_VALUE,
} from "utilities/constants";

import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";

const baseClass = "host-summary-card";

interface IBootstrapPackageData {
  status?: BootstrapPackageStatus | "";
  details?: string;
}

interface IHostSummaryProps {
  summaryData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  bootstrapPackageData?: IBootstrapPackageData;
  isPremiumTier?: boolean;
  toggleBootstrapPackageModal?: () => void;
  toggleOnlineHistoryModal?: () => void;
  className?: string;
}

const HostSummary = ({
  summaryData,
  bootstrapPackageData,
  isPremiumTier,
  toggleBootstrapPackageModal,
  toggleOnlineHistoryModal,
  className,
}: IHostSummaryProps): JSX.Element | null => {
  const classNames = classnames(baseClass, className);

  const { status, platform, mdm } = summaryData;

  const isAndroidHost = isAndroid(platform);
  const isIosOrIpadosHost = isIPadOrIPhone(platform);

  const renderIssues = () => (
    <DataSet
      title="Issues"
      value={
        <IssuesIndicator
          totalIssuesCount={summaryData.issues.total_issues_count}
          criticalVulnerabilitiesCount={
            summaryData.issues.critical_vulnerabilities_count
          }
          failingPoliciesCount={summaryData.issues.failing_policies_count}
          tooltipPosition="bottom"
        />
      }
    />
  );

  const renderHostTeam = () => (
    <DataSet
      title="Fleet"
      value={
        summaryData.team_name !== DEFAULT_EMPTY_CELL_VALUE ? (
          `${summaryData.team_name}`
        ) : (
          <span className="no-team">Unassigned</span>
        )
      }
    />
  );

  const renderMaintenanceWindow = ({
    starts_at,
    timezone,
  }: IHostMaintenanceWindow) => {
    const formattedStartsAt = formatInTimeZone(
      starts_at,
      // since startsAt is already localized and contains offset information, this 2nd parameter is
      // logically redundant. It's included here to allow use of date-fns-tz.formatInTimeZone instead of date-fns.format, which
      // allows us to format a UTC datetime without converting to the user-agent local time.
      timezone || "UTC",
      DATE_FNS_FORMAT_STRINGS.dateAtTime
    );

    const tip =
      timezone && timezone !== "UTC" ? (
        <>
          End user&apos;s time zone:
          <br />
          (GMT{starts_at.slice(-6)}) {timezone.replace("_", " ")}
        </>
      ) : (
        <>
          End user&apos;s timezone unavailable.
          <br />
          Displaying in UTC.
        </>
      );

    return (
      <DataSet
        title="Scheduled maintenance"
        value={
          <TooltipWrapper tipContent={tip}>{formattedStartsAt}</TooltipWrapper>
        }
      />
    );
  };

  const showStatus = !isIosOrIpadosHost && !isAndroidHost;
  const showTeam = !!isPremiumTier;
  const showIssues =
    summaryData.issues?.total_issues_count > 0 &&
    !isIosOrIpadosHost &&
    !isAndroidHost;
  const showBootstrapPackage =
    !!bootstrapPackageData?.status && !isIosOrIpadosHost && !isAndroidHost;
  const showMaintenanceWindow =
    !!isPremiumTier &&
    // TODO - refactor normalizeEmptyValues pattern
    !!summaryData.maintenance_window &&
    summaryData.maintenance_window !== DEFAULT_EMPTY_CELL_VALUE;
  // Mobile hosts have no status pill; instead the card exposes a Status row
  // with a "View history" action so users can still see connectivity trends.
  const showMobileOnlineHistoryRow =
    !!toggleOnlineHistoryModal && (isIosOrIpadosHost || isAndroidHost);

  // Hide the card entirely when nothing inside it would render (e.g. a Free
  // tier Android host with no online-history entry point) — otherwise an
  // empty card sits above the Vitals section.
  if (
    !showStatus &&
    !showTeam &&
    !showIssues &&
    !showBootstrapPackage &&
    !showMaintenanceWindow &&
    !showMobileOnlineHistoryRow
  ) {
    return null;
  }

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      className={classNames}
    >
      {showStatus && (
        <DataSet
          title="Status"
          value={
            toggleOnlineHistoryModal ? (
              <button
                type="button"
                className={`${baseClass}__status-button`}
                onClick={toggleOnlineHistoryModal}
                aria-label="View online history"
              >
                <StatusIndicator
                  value={getHostStatus(status, mdm?.enrollment_status)}
                />
              </button>
            ) : (
              <StatusIndicator
                value={getHostStatus(status, mdm?.enrollment_status)}
                tooltip={{
                  tooltipText: getHostStatusTooltipText(
                    getHostStatus(status, mdm?.enrollment_status)
                  ),
                  position: "bottom",
                }}
              />
            )
          }
        />
      )}
      {showMobileOnlineHistoryRow && (
        <DataSet
          title={
            isIosOrIpadosHost ? (
              <TooltipWrapper
                tipContent="iOS/iPadOS hosts are online anytime they have power and an internet connection (including locked)."
                position="top"
                showArrow
                tipOffset={8}
              >
                Status
              </TooltipWrapper>
            ) : (
              "Status"
            )
          }
          value={
            <Button
              variant="link"
              onClick={toggleOnlineHistoryModal}
              className={`${baseClass}__view-history-button`}
            >
              View history
            </Button>
          }
        />
      )}
      {showTeam && renderHostTeam()}
      {showIssues && renderIssues()}
      {showBootstrapPackage && bootstrapPackageData?.status && (
        <DataSet
          title="Bootstrap package"
          value={
            <BootstrapPackageIndicator
              status={bootstrapPackageData.status}
              onClick={toggleBootstrapPackageModal}
            />
          }
        />
      )}
      {showMaintenanceWindow &&
        renderMaintenanceWindow(summaryData.maintenance_window)}
    </Card>
  );
};

export default HostSummary;
