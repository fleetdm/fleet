import React, { useEffect, useState } from "react";
import { Command } from "cmdk";
import { useQuery } from "react-query";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import queriesAPI, { IQueriesResponse } from "services/entities/queries";

const baseClass = "command-palette";

const REPORT_SEARCH_LIMIT = 50;
const REPORT_SEARCH_DEBOUNCE_MS = 200;

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
  const [debouncedQuery, setDebouncedQuery] = useState(search.trim());
  useEffect(() => {
    const id = window.setTimeout(() => {
      setDebouncedQuery(search.trim());
    }, REPORT_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [search]);

  // Scope reports to the currently selected fleet. mergeInherited so team
  // views surface inherited global reports too.
  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;
  const fleetLabel =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.name
      : "All fleets";

  const { data, isLoading } = useQuery<IQueriesResponse, Error>(
    ["commandPaletteReports", teamId, debouncedQuery],
    () =>
      queriesAPI.loadAll({
        teamId,
        page: 0,
        perPage: REPORT_SEARCH_LIMIT,
        query: debouncedQuery || undefined,
        orderKey: "name",
        orderDirection: "asc",
        mergeInherited: true,
      }),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  const reports = data?.queries ?? [];

  if (isLoading && reports.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for reports...</div>;
  }

  if (reports.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>
        {debouncedQuery
          ? `No reports match "${debouncedQuery}" in ${fleetLabel}.`
          : `No reports found in ${fleetLabel}.`}
      </div>
    );
  }

  return (
    <Command.Group className={`${baseClass}__group`}>
      {reports.map((report) => (
        <Command.Item
          key={`report-${report.id}`}
          value={`REPORT_RESULT ${report.id}`}
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
