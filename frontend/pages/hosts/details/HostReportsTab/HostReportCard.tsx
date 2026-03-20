import React, { useCallback, useMemo } from "react";

import { IHostReport } from "services/entities/host_reports";
import { humanLastSeen } from "utilities/helpers";

import Button from "components/buttons/Button";
import Card from "components/Card";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import ActionsDropdown from "components/ActionsDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";
import PillBadge from "components/PillBadge";
import TooltipTruncatedText from "components/TooltipTruncatedText";

const baseClass = "host-report-card";
const iconColor = "ui-fleet-black-75";

interface IHostReportCardProps {
  report: IHostReport;
  hostName: string;
  onShowDetails: (report: IHostReport) => void;
  onViewAllHosts: (report: IHostReport) => void;
}

const HostReportCard = ({
  report,
  hostName,
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

  const actionOptions: IDropdownOption[] = useMemo(() => {
    const options: IDropdownOption[] = [];
    if (hasResults) {
      options.push({ value: "show_details", label: "Show details" });
    }
    options.push({
      value: "view_all_hosts",
      label: "View report for all hosts",
    });
    return options;
  }, [hasResults]);

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
          <DataSet
            key={key}
            title={key}
            value={<TooltipTruncatedText value={value} />}
          />
        ))}
      </div>
    );
  };

  const renderBanner = () => {
    // Report doesn't store results
    if (doesNotStoreResults) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <div className={`${baseClass}__banner-text`}>
              <Icon name="info-outline" color={iconColor} />
              Results from this report are not stored in Fleet.
            </div>
          </div>
        </InfoBanner>
      );
    }

    // Awaiting results
    if (isAwaitingResults) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <div className={`${baseClass}__banner-text`}>
              <Icon name="pending-outline" color={iconColor} />
              Fleet is awaiting results from {hostName}.
            </div>
          </div>
        </InfoBanner>
      );
    }

    // Has run but returned no data
    if (!hasData) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <div className={`${baseClass}__banner-text`}>
              <Icon name="check" color={iconColor} />
              This report has run on {hostName}, but returned no data for this
              host.
            </div>
          </div>
        </InfoBanner>
      );
    }

    // Has data and additional results
    if (report.n_host_results > 1) {
      return (
        <InfoBanner color="grey" borderRadius="xlarge">
          <div className={`${baseClass}__banner-content`}>
            <div className={`${baseClass}__banner-text`}>
              <Icon name="info-outline" color={iconColor} />
              {report.n_host_results - 1} additional result
              {report.n_host_results - 1 !== 1 ? "s" : ""} not shown
            </div>
            <Button
              className={`${baseClass}__view-full-report`}
              variant="inverse"
              size="small"
              onClick={() => onShowDetails(report)}
            >
              View full report
              <Icon name="chevron-right" color={iconColor} />
            </Button>
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
            <PillBadge
              className={`${baseClass}__clipped-badge`}
              tipContent="This report has paused saving results. If automations are enabled, results are still sent to your log destination."
            >
              <Icon size="small" name="warning" color="ui-fleet-black-75" />
              Report clipped
            </PillBadge>
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
