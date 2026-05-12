import React, { ReactNode, useCallback, useMemo } from "react";

import { IHostReport } from "services/entities/host_reports";
import { humanLastSeen } from "utilities/helpers";
import { pluralize } from "utilities/strings/stringUtils";

import Button from "components/buttons/Button";
import Card from "components/Card";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import InfoBanner from "components/InfoBanner";
import ActionsDropdown from "components/ActionsDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";
import PillBadge from "components/PillBadge";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { Colors } from "styles/var/colors";

const baseClass = "host-report-card";
const ICON_COLOR: Colors = "ui-fleet-black-75";

const ACTION_SHOW_DETAILS = "show_details";
const ACTION_VIEW_ALL_HOSTS = "view_all_hosts";

const ReportBanner = ({
  iconName,
  message,
  children,
}: {
  iconName: IconNames;
  message: ReactNode;
  children?: ReactNode;
}) => (
  <InfoBanner borderRadius="xlarge">
    <div className={`${baseClass}__banner-content`}>
      <div className={`${baseClass}__banner-text`}>
        <Icon name={iconName} color={ICON_COLOR} />
        {message}
      </div>
      {children}
    </div>
  </InfoBanner>
);

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
      if (value === ACTION_SHOW_DETAILS) {
        onShowDetails(report);
      } else if (value === ACTION_VIEW_ALL_HOSTS) {
        onViewAllHosts(report);
      }
    },
    [report, onShowDetails, onViewAllHosts]
  );

  const actionOptions: IDropdownOption[] = useMemo(() => {
    const options: IDropdownOption[] = [];
    if (hasResults) {
      options.push({ value: ACTION_SHOW_DETAILS, label: "Show details" });
    }
    options.push({
      value: ACTION_VIEW_ALL_HOSTS,
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
            textOnly
          />
        ))}
      </div>
    );
  };

  const renderBanner = () => {
    if (doesNotStoreResults) {
      return (
        <ReportBanner
          iconName="info-outline"
          message="Results from this report are not stored in Fleet."
        />
      );
    }

    if (isAwaitingResults) {
      return (
        <ReportBanner
          iconName="pending-outline"
          message={`Fleet is awaiting results from ${hostName}.`}
        />
      );
    }

    if (!hasData) {
      return (
        <ReportBanner
          iconName="check"
          message={`This report has run on ${hostName}, but returned no data for this host.`}
        />
      );
    }

    if (report.n_host_results > 1) {
      const additionalCount = report.n_host_results - 1;
      return (
        <ReportBanner
          iconName="info-outline"
          message={`${additionalCount} additional ${pluralize(
            additionalCount,
            "result"
          )} not shown`}
        >
          <Button
            className={`${baseClass}__view-full-report`}
            variant="inverse"
            size="small"
            onClick={() => onShowDetails(report)}
          >
            View full report
            <Icon name="chevron-right" color={ICON_COLOR} />
          </Button>
        </ReportBanner>
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
              <Icon size="small" name="warning" color={ICON_COLOR} />
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
