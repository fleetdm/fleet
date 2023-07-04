import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";

const baseClass = "schedule";

interface IScheduleProps {
  schedule?: IQueryStats[];
  isChromeOSHost: boolean;
  isLoading: boolean;
}

const Schedule = ({
  schedule,
  isChromeOSHost,
  isLoading,
}: IScheduleProps): JSX.Element => {
  const wrapperClassName = `${baseClass}__pack-table`;
  const tableHeaders = generateTableHeaders();

  const renderEmptyScheduleTab = () => {
    if (isChromeOSHost) {
      return (
        <EmptyTable
          header="Scheduled queries are not supported for this host"
          info={
            <>
              <span>Interested in collecting data from your Chromebooks? </span>
              <CustomLink
                url="https://www.fleetdm.com/contact"
                text="Let us know"
                newTab
              />
            </>
          }
        />
      );
    }
    return (
      <EmptyTable
        header="No queries are scheduled for this host"
        info="Expecting to see queries? Try selecting “Refetch” to ask this host
            to report new vitals."
      />
    );
  };

  return (
    <div className="section section--schedule">
      <p className="section__header">Schedule</p>
      {!schedule || !schedule.length || isChromeOSHost ? (
        renderEmptyScheduleTab()
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
