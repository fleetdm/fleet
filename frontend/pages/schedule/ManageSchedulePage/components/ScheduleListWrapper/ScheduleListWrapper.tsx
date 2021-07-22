/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React, { useState, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";

import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";

import TableContainer from "components/TableContainer";
import { generateTableHeaders } from "./ScheduleTableConfig";
import NoSchedule from "../NoSchedule";

const baseClass = "schedule-list-wrapper";

interface IScheduleListWrapperProps {
  onRemoveScheduledQueryClick: any;
  allGlobalScheduledQueriesList: IGlobalScheduledQuery[];
  toggleScheduleEditorModal: any;
}

interface INoScheduledQueriesProps {
  toggleScheduleEditorModal: any;
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
  const {
    onRemoveScheduledQueryClick,
    allGlobalScheduledQueriesList,
    toggleScheduleEditorModal,
  } = props;
  const dispatch = useDispatch();

  console.log("schedulelistwrapper", typeof toggleScheduleEditorModal);

  // Hardcode in needed props
  const onActionSelection = () => null;

  const NoScheduledQueries = (): JSX.Element => {
    console.log("noscheduledqueries", typeof toggleScheduleEditorModal);

    return <NoSchedule toggleScheduleEditorModal={toggleScheduleEditorModal} />;
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.global_scheduled_queries.isLoading
  );

  // Search functionality disabled, needed if enabled
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
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={allGlobalScheduledQueriesList}
        isLoading={loadingTableData}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages
        isAllPagesSelected
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search"
        searchable={false}
        onSelectActionClick={onRemoveScheduledQueryClick}
        selectActionButtonText={"Remove query"}
        emptyComponent={NoScheduledQueries} // this empty component needed a togglefunction passed in, my fix is super janky
      />
    </div>
  );
};

export default ScheduleListWrapper;
