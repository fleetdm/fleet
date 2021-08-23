/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React, { useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";
import { push } from "react-router-redux";
import paths from "router/paths";

import Button from "components/buttons/Button";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import { ITeamScheduledQuery } from "interfaces/team_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";

import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./ScheduleTableConfig";
// @ts-ignore
import scheduleSvg from "../../../../../../assets/images/schedule.svg";

const baseClass = "schedule-list-wrapper";
const noScheduleClass = "no-schedule";

interface IScheduleListWrapperProps {
  onRemoveScheduledQueryClick: any;
  onEditScheduledQueryClick: any;
  allScheduledQueriesList: IGlobalScheduledQuery[] | ITeamScheduledQuery[];
  toggleScheduleEditorModal: () => void;
  teamId: number;
}
interface IRootState {
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
    };
    team_scheduled_queries: {
      isLoading: boolean;
      data: ITeamScheduledQuery[];
    };
  };
}

const ScheduleListWrapper = (props: IScheduleListWrapperProps): JSX.Element => {
  const {
    onRemoveScheduledQueryClick,
    allScheduledQueriesList,
    toggleScheduleEditorModal,
    onEditScheduledQueryClick,
    teamId,
  } = props;
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;

  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  const NoScheduledQueries = () => {
    return (
      <div className={`${noScheduleClass}`}>
        <div className={`${noScheduleClass}__inner`}>
          <img src={scheduleSvg} alt="No Schedule" />
          <div className={`${noScheduleClass}__inner-text`}>
            <h2>You don&apos;t have any queries scheduled.</h2>
            <p>
              Schedule a query, or go to your osquery packs via the
              &lsquo;Advanced&rsquo; button.
            </p>
            <div className={`${noScheduleClass}__-cta-buttons`}>
              <Button
                variant="brand"
                className={`${noScheduleClass}__schedule-button`}
                onClick={toggleScheduleEditorModal}
              >
                Schedule a query
              </Button>
              <Button
                variant="inverse"
                onClick={handleAdvanced}
                className={`${baseClass}__advanced-button`}
              >
                Advanced
              </Button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const onActionSelection = (
    action: string,
    global_scheduled_query: IGlobalScheduledQuery
  ): void => {
    switch (action) {
      case "edit":
        onEditScheduledQueryClick(global_scheduled_query);
        break;
      default:
        onRemoveScheduledQueryClick([global_scheduled_query.id]);
        break;
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const loadingTableData = useSelector((state: IRootState) => {
    if (teamId) {
      return state.entities.team_scheduled_queries.isLoading;
    }
    return state.entities.global_scheduled_queries.isLoading;
  });

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
        data={generateDataSet(allScheduledQueriesList, teamId)}
        isLoading={loadingTableData}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search"
        searchable={false}
        disablePagination
        onPrimarySelectActionClick={onRemoveScheduledQueryClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="close"
        primarySelectActionButtonText={"Remove"}
        emptyComponent={NoScheduledQueries}
      />
    </div>
  );
};

export default ScheduleListWrapper;
