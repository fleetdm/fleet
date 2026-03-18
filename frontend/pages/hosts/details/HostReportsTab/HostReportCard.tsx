import React, { useCallback } from "react";

import { IHostReport } from "services/entities/host_reports";
import { humanLastSeen } from "utilities/helpers";

import Card from "components/Card";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import ActionsDropdown from "components/ActionsDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";
import TooltipWrapper from "components/TooltipWrapper";
import PATHS from "router/paths";

const baseClass = "host-report-card";

interface IHostReportCardProps {
  report: IHostReport;
  hostName: string;
  hostId: number;
  onShowDetails: (report: IHostReport) => void;
  onViewAllHosts: (report: IHostReport) => void;
}

const HostReportCard = ({
  report,
  hostName,
  hostId,
  onShowDetails,
  onViewAllHosts,
}: IHostReportCardProps): JSX.Element => {
  const hasResults = report.last_fetched !== null;
  const hasData = report.first_result !== null;
  const isAwaitingResults = !hasResults;
  const doesNotStoreResults = !report.store_results;

  const onActionChange = useCallback(
    (value: string) => {
      if (value === "show_details") {
        onShowDetails(report);
      } else if (value === "view_all_hosts") {
        onViewAllHosts(report);
      }
    },
    [report, onShowDetails, onViewAllHosts]
  );

  const actionOptions: IDropdownOption[] = [];
  if (hasResults) {
    actionOptions.push({
      value: "show_details",
      label: "Show details",
    });
  }
  actionOptions.push({
    value: "view_all_hosts",
    label: "View report for all hosts",
  });

  const renderLastUpdated = () => {
    if (isAwaitingResults) return null;

    const prefix = doesNotStoreResults ? "Last ran" : "Last updated";
    return (
      <span className={`${baseClass}__last-updated`}>
        {prefix} {humanLastSeen(report.last_fetched || "")}
      </span>
    );
  };

  const renderDataCells = () => {
    if (!hasData || !report.first_result) return null;

    const entries = Object.entries(report.first_result);
    return (
      <div className={`${baseClass}__data-grid`}>
        {entries.map(([key, value]) => (
          <div key={key} className={`${baseClass}__data-cell`}>
            <span className={`${baseClass}__data-key`}>{key}</span>
            <span className={`${baseClass}__data-value`}>{value}</span>
          </div>
        ))}
      </div>
    );
  };

  const renderBanner = () => {
    // Scenario 5: Report doesn't store results
    if (doesNotStoreResults) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <Icon name="info" />
            Results from this report are not stored in Fleet.
          </div>
        </InfoBanner>
      );
    }

    // Scenario 4: Awaiting results
    if (isAwaitingResults) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <Icon name="more" />
            Fleet is awaiting results from {hostName}.
          </div>
        </InfoBanner>
      );
    }

    // Scenario 3: Has run but returned no data
    if (!hasData) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <Icon name="check" />
            This report has run on {hostName}, but returned no data for this
            host.
          </div>
        </InfoBanner>
      );
    }

    // Scenario 2: Has data and additional results
    if (report.n_host_results > 1) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <Icon name="info" />
            <span>
              {report.n_host_results - 1} additional result
              {report.n_host_results - 1 !== 1 ? "s" : ""} not shown
            </span>
            <a
              href={PATHS.HOST_REPORT_RESULTS(hostId, report.query_id)}
              className={`${baseClass}__view-full-report`}
            >
              View full report &gt;
            </a>
          </div>
        </InfoBanner>
      );
    }

    return null;
  };

  return (
    <Card className={baseClass} borderRadiusSize="xlarge" paddingSize="xlarge">
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header-left`}>
          <div className={`${baseClass}__title-row`}>
            <h3 className={`${baseClass}__name`}>{report.name}</h3>
            {renderLastUpdated()}
          </div>
          {report.description && (
            <p className={`${baseClass}__description`}>{report.description}</p>
          )}
        </div>
        <div className={`${baseClass}__header-right`}>
          {report.report_clipped && (
            <TooltipWrapper
              tipContent="This report has paused saving results. If automations are enabled, results are still sent to your log destination."
              showArrow
              position="top"
            >
              <span className={`${baseClass}__clipped-badge`}>
                <Icon name="warning" />
                Report clipped
              </span>
            </TooltipWrapper>
          )}
          <ActionsDropdown
            options={actionOptions}
            placeholder="Actions"
            onChange={onActionChange}
            variant="button"
            menuAlign="right"
          />
        </div>
      </div>

      {renderDataCells()}
      {renderBanner()}
    </Card>
  );
};

export default HostReportCard;
