import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import TableContainer from "components/TableContainer";

import {
  generatePackTableHeaders,
  generatePackDataSet,
} from "../Packs/PackTable/PackTableConfig";

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
  const tableHeaders = generatePackTableHeaders();

  return (
    <div className="section section--packs">
      <p className="section__header">Schedule</p>
      {!schedule || !schedule.length ? (
        <div className="results__data">
          <b>No queries are scheduled for this host.</b>
          <p>
            Expecting to see queries? Try selecting “Refetch” to ask this host
            to report new vitals.
          </p>
        </div>
      ) : (
        <div className={`${wrapperClassName}`}>
          <TableContainer
            columns={tableHeaders}
            data={generatePackDataSet(schedule)}
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
