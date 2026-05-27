import React from "react";
import { Command } from "cmdk";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import queriesAPI, { IQueriesResponse } from "services/entities/queries";
import { ISchedulableQuery } from "interfaces/schedulable_query";

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";
import { getFleetSuffix } from "./pickerCopy";

const baseClass = "command-palette";

const REPORT_SEARCH_LIMIT = 50;

interface IReportPickerProps {
  search: string;
  currentTeam?: ITeamSummary;
  onSelect: (reportId: number) => void;
}

const ReportPicker = ({
  search,
  currentTeam,
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

  return (
    <Command.Group className={`${baseClass}__group`}>
      {reports.map((report) => (
        <Command.Item
          key={`report-${report.id}`}
          value={`${RESULT_PREFIXES.report}${report.id}`}
          onSelect={() => onSelect(report.id)}
          className={`${baseClass}__item`}
        >
          <span className={`${baseClass}__item-label`}>{report.name}</span>
        </Command.Item>
      ))}
    </Command.Group>
  );
};

export default ReportPicker;
