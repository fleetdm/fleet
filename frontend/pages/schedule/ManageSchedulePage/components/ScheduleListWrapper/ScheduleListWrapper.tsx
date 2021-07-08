/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React, { useState, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";

// @ts-ignore
import queriesActions from "redux/nodes/entities/queries/actions";

import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";
import NoSchedule from "../NoSchedule";

const baseClass = "schedule-list-wrapper";

interface IScheduleListWrapperProps {
  fakeData: any;
  toggleRemoveScheduledQueryModal: any;
}

const ScheduleListWrapper = (props: IScheduleListWrapperProps): JSX.Element => {
  const { fakeData, toggleRemoveScheduledQueryModal } = props;
  const dispatch = useDispatch();

  const scheduledCount = fakeData.scheduled.length;
  const scheduledQueriesTotalDisplay =
    scheduledCount === 1 ? "1 query" : `${scheduledCount} queries`;

  // Hardcode in needed props
  const loadingQueries = false;
  const onActionSelection = () => null;

  const tableHeaders = generateTableHeaders(onActionSelection);

  // Query ID state crap
  // What happens when you do a click whatever whatever
  const [selectedQueryIds, setSelectedQueryIds] = useState([]);

  // Remove query button
  const onRemoveScheduledQueryClick = (selectedQueryIds: any) => {
    console.log(
      "onRemoveScheduledQueryClick from ScheduleListWrapper fires when clicking Remove queries in the table. It toggles the modal and sets the selectedQuery ids."
    );
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedQueryIds);
  };

  // Search functionality
  // NOTE: called once on the initial render of this component.
  const onQueryChange = useCallback(
    (queryData) => {
      const { pageIndex, pageSize, searchQuery } = queryData;
      dispatch(
        queriesActions.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
        })
      );
    },
    [dispatch]
  );

  return (
    <div className={`${baseClass}`}>
      {/* Not using this because the search functionality returns a count
      <p className={`${baseClass}__scheduled-query-count`}>
        {scheduledQueriesTotalDisplay}
      </p> */}
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={fakeData.scheduled}
        // TODO: connect loading state to this table
        isLoading={loadingQueries}
        // Removed action button next to search
        // actionButtonText={"Remove query"}
        // actionButtonVariant={"primary"}
        // onActionButtonClick={toggleRemoveScheduledQueryModal}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        // TODO: QA search functionality later
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search"
        // TODO: figure out toggle for onSelectActionClick
        onSelectActionClick={onRemoveScheduledQueryClick}
        selectActionButtonText={"Remove query"}
        emptyComponent={NoSchedule}
      />
    </div>
  );
};

export default ScheduleListWrapper;
