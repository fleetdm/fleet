/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React, { useState, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";

import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";

import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";
import NoSchedule from "../NoSchedule";

const baseClass = "schedule-list-wrapper";

interface IScheduleListWrapperProps {
  fakeData: any;
  toggleRemoveScheduledQueryModal: any;
}
interface IRootState {
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
    };
  };
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

  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.global_scheduled_queries.isLoading
  );
  const teams = useSelector((state: IRootState) =>
    generateDataSet(state.entities.global_scheduled_queries.data)
  );

  // The state of the selectedQueryIds changes when you click Remove query button
  const [selectedQueryIds, setSelectedQueryIds] = useState([]);

  // Table CTA: Remove query button
  const onRemoveScheduledQueryClick = (selectedQueryIds: any) => {
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedQueryIds);
  };

  // Search functionality disabled, needed if enabled
  // NOTE: called once on the initial render of this component.
  const onQueryChange = useCallback(
    (queryData) => {
      const { pageIndex, pageSize, searchQuery } = queryData;
      dispatch(
        globalScheduledQueryActions.loadAll({
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
        // Removed search functionality
        showMarkAllPages
        isAllPagesSelected
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search"
        searchable={false}
        onSelectActionClick={onRemoveScheduledQueryClick}
        selectActionButtonText={"Remove query"}
        emptyComponent={NoSchedule}
      />
    </div>
  );
};

export default ScheduleListWrapper;
