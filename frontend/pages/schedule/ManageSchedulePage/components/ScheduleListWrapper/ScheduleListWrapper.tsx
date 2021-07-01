/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React from "react";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";

const baseClass = "schedule-list-wrapper";

interface IScheduleListWrapperProps {
  fakeData: any;
}

const ScheduleListWrapper = (props: IScheduleListWrapperProps): JSX.Element => {
  const { fakeData } = props;

  const scheduledCount = fakeData.scheduled.length;
  const scheduledQueriesTotalDisplay =
    scheduledCount === 1 ? "1 query" : `${scheduledCount} queries`;

  // const tableHeaders = generateTableHeaders(onActionSelection);

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__scheduled-query-count`}>
        {scheduledQueriesTotalDisplay}
      </p>
      Hello Schedule Table
      <TableContainer
        // TODO: Figure out how to TableConfig table headers
        columns={tableHeaders}
        data={fakeData}
        // TODO: connect loading state to this table
        isLoading={loadingQueries}
        actionButtonText={"Delete Query"}
        actionButtonVariant={"primary"}
        // TODO: build delete query modal
        onActionButtonClick={toggleDeleteQueryModal}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        // TODO: figure out what this does
        onSelectActionClick={onDeleteQueryClick}
        // TODO: figure out what this should look like
        emptyComponent={EmptyQueries}
      />
    </div>
  );
};

export default ScheduleListWrapper;
