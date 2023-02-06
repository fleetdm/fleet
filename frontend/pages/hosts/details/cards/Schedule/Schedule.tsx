import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";

const baseClass = "schedule";

interface IScheduleProps {
  scheduleState?: IQueryStats[];
  isLoading: boolean;
}

const Schedule = ({
  scheduleState,
  isLoading,
}: IScheduleProps): JSX.Element => {
  const schedule = scheduleState;
  const wrapperClassName = `${baseClass}__pack-table`;
  const tableHeaders = generateTableHeaders();

  return (
    <div className="section section--schedule">
      <p className="section__header">Schedule</p>
      {!schedule || !schedule.length ? (
        <EmptyTable
          header="No queries are scheduled for this host"
          info="Expecting to see queries? Try selecting “Refetch” to ask this host
            to report new vitals."
        />
      ) : (
        <div className={`${wrapperClassName}`}>
          <TableContainer
            columns={tableHeaders}
            data={generateDataSet(schedule)}
            isLoading={isLoading}
            onQueryChange={() => null}
            resultsTitle={"queries"}
            defaultSortHeader={"scheduled_query_name"}
            defaultSortDirection={"asc"}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            emptyComponent={() => <></>}
            disablePagination
            disableCount
          />
        </div>
      )}
    </div>
  );
};

export default Schedule;
