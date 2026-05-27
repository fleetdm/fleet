import React from "react";
import { Command } from "cmdk";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import queriesAPI, { IQueriesResponse } from "services/entities/queries";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import Icon from "components/Icon";
import PillBadge from "components/PillBadge";
import TooltipWrapper from "components/TooltipWrapper";

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";
import getFleetSuffix from "./pickerCopy";

const baseClass = "command-palette";

const REPORT_SEARCH_LIMIT = 50;

interface IReportPickerProps {
  search: string;
  currentTeam?: ITeamSummary;
  /** True when the viewer is an observer in the current team scope.
   *  Suppresses the "Observers can run" indicator on those reports — the
   *  badge is meant to flag reports that *non-observers* can hand off to
   *  observers, not to advertise the current user's own capability. */
  isViewerObserver?: boolean;
  onSelect: (reportId: number) => void;
}

const ReportPicker = ({
  search,
  currentTeam,
  isViewerObserver = false,
  onSelect,
}: IReportPickerProps): JSX.Element => {
  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;

  const fleetSuffix = getFleetSuffix(currentTeam);

  const { items: reports, isLoading, debouncedQuery } = usePickerSearch<
    IQueriesResponse,
    ISchedulableQuery
  >({
    search,
    queryKeyPrefix: ["commandPaletteReports", teamId ?? "global"],
    queryFn: (q) =>
      queriesAPI.loadAll({
        scope: "queries",
        teamId,
        page: 0,
        perPage: REPORT_SEARCH_LIMIT,
        query: q || undefined,
        orderKey: "name",
        orderDirection: "asc",
        mergeInherited: true,
      }),
    selectItems: (data) => data?.queries ?? [],
  });

  if (isLoading && reports.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for reports...</div>;
  }

  if (reports.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>
        {debouncedQuery
          ? `No reports match "${debouncedQuery}"${fleetSuffix}.`
          : `No reports found${fleetSuffix}.`}
      </div>
    );
  }

  // "Inherited" applies only when viewing a specific team and the report
  // lives at a different scope (typically global, team_id null/different).
  const isViewingSpecificTeam =
    !!currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID;

  return (
    <Command.Group className={`${baseClass}__group`}>
      {reports.map((report) => {
        const showObserverIcon = !isViewerObserver && report.observer_can_run;
        const showInheritedBadge =
          isViewingSpecificTeam && report.team_id !== currentTeam?.id;

        return (
          <Command.Item
            key={`report-${report.id}`}
            value={`${RESULT_PREFIXES.report}${report.id}`}
            onSelect={() => onSelect(report.id)}
            className={`${baseClass}__item`}
          >
            <div className={`${baseClass}__item-left`}>
              <span className={`${baseClass}__item-label`}>{report.name}</span>
              {showObserverIcon && (
                <TooltipWrapper
                  tipContent="Observers can run this report."
                  underline={false}
                  showArrow
                  position="top"
                  delayInMs={300}
                >
                  <Icon name="query" size="small" color="ui-fleet-black-50" />
                </TooltipWrapper>
              )}
              {showInheritedBadge && (
                <PillBadge tipContent="This report runs on all hosts.">
                  Inherited
                </PillBadge>
              )}
            </div>
          </Command.Item>
        );
      })}
    </Command.Group>
  );
};

export default ReportPicker;
