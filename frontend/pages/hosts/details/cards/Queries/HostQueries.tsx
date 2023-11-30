import React from "react";

import { IQueryStats } from "interfaces/query_stats";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

import {
  generateTableHeaders,
  generateDataSet,
} from "./HostQueriesTableConfig";

const baseClass = "host-queries";

interface IHostQueriesProps {
  schedule?: IQueryStats[];
  isChromeOSHost: boolean;
  isLoading: boolean;
}

const HostQueries = ({
  schedule,
  isChromeOSHost,
  isLoading,
}: IHostQueriesProps): JSX.Element => {
  const tableHeaders = generateTableHeaders();

  const renderEmptyQueriesTab = () => {
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
        header="No queries are scheduled to run on this host"
        info={
          <>
            Expecting to see queries? Try selecting <b>Refetch</b> to ask this
            host to report fresh vitals.
          </>
        }
      />
    );
  };

  return (
    <div className="section section--host-queries">
      <p className="section__header">Queries</p>
      {!schedule || !schedule.length || isChromeOSHost ? (
        renderEmptyQueriesTab()
      ) : (
        <div className={`${baseClass}__pack-table`}>
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

export default HostQueries;
