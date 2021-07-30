/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React, { useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";
import { push } from "react-router-redux";
import paths from "router/paths";

import { IQuery } from "interfaces/query";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions";

import TableContainer from "components/TableContainer";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-list-wrapper";
const noQueriesClass = "no-queries";

interface IQueriesListWrapperProps {
  onRemoveQueryClick: any;
  queriesList: IQuery[];
  // toggleScheduleEditorModal: any;
}
interface IRootState {
  entities: {
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
  };
}

const QueriesListWrapper = (props: IQueriesListWrapperProps): JSX.Element => {
  const {
    onRemoveQueryClick,
    queriesList,
    // toggleScheduleEditorModal,
  } = props;
  const dispatch = useDispatch();

  const NoScheduledQueries = () => {
    return (
      <div className={`${noQueriesClass}`}>
        <div className={`${noQueriesClass}__inner`}>
          {/* <img src={scheduleSvg} alt="No Schedule" /> */}
          <div className={`${noQueriesClass}__inner-text`}>
            <h2>You don&apos;t have any queries.</h2>
            <p>
              Create a new query, or{" "}
              <a href="https://github.com/fleetdm/fleet/tree/main/docs/1-Using-Fleet/standard-query-library">
                go to GitHub
              </a>{" "}
              to import Fleet’s standard query library.
            </p>
          </div>
        </div>
      </div>
    );
  };

  const tableHeaders = generateTableHeaders();
  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.queries.isLoading
  );

  // Search functionality disabled, needed if enabled
  const onQueryChange = useCallback(
    (queryData) => {
      const { pageIndex, pageSize, searchQuery } = queryData;
      dispatch(
        queryActions.loadAll({
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
        data={queriesList}
        isLoading={loadingTableData}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable
        disablePagination
        onPrimarySelectActionClick={onRemoveQueryClick}
        primarySelectActionButtonVariant="text-link"
        primarySelectActionButtonIcon="close"
        primarySelectActionButtonText={"Remove"}
        emptyComponent={NoScheduledQueries}
      />
    </div>
  );
};

export default QueriesListWrapper;
