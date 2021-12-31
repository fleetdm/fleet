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
import { ITeam } from "interfaces/team";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";

import TableContainer from "components/TableContainer";
import {
  generateInheritedQueriesTableHeaders,
  generateTableHeaders,
  generateDataSet,
} from "./ScheduleTableConfig";
// @ts-ignore
import scheduleSvg from "../../../../../../assets/images/no-schedule-322x138@2x.png";

const baseClass = "schedule-list-wrapper";
const noScheduleClass = "no-schedule";

const TAGGED_TEMPLATES = {
  hostsByTeamPRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};
interface IScheduleListWrapperProps {
  onRemoveScheduledQueryClick?: (selectIds: number[]) => void;
  onEditScheduledQueryClick?: (
    selectedQuery: IGlobalScheduledQuery | ITeamScheduledQuery
  ) => void;
  allScheduledQueriesList: IGlobalScheduledQuery[] | ITeamScheduledQuery[];
  toggleScheduleEditorModal?: () => void;
  inheritedQueries?: boolean;
  isOnGlobalTeam: boolean;
  selectedTeamData: ITeam | undefined;
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

const ScheduleListWrapper = ({
  onRemoveScheduledQueryClick,
  allScheduledQueriesList,
  toggleScheduleEditorModal,
  onEditScheduledQueryClick,
  inheritedQueries,
  isOnGlobalTeam,
  selectedTeamData,
}: IScheduleListWrapperProps): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS, MANAGE_HOSTS } = paths;

  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  const NoScheduledQueries = () => {
    return (
      <div
        className={`${noScheduleClass} ${
          selectedTeamData?.id && "no-team-schedule"
        }`}
      >
        <div className={`${noScheduleClass}__inner`}>
          <img src={scheduleSvg} alt="No Schedule" />
          <div className={`${noScheduleClass}__inner-text`}>
            <p>
              <b>
                {selectedTeamData ? (
                  <>
                    Schedule queries for all hosts assigned to{" "}
                    <a
                      href={
                        MANAGE_HOSTS +
                        TAGGED_TEMPLATES.hostsByTeamPRoute(selectedTeamData.id)
                      }
                    >
                      {selectedTeamData.name}
                    </a>
                  </>
                ) : (
                  <>
                    Schedule queries to run at regular intervals on{" "}
                    <a href={MANAGE_HOSTS}>all your hosts</a>
                  </>
                )}
              </b>
              {isOnGlobalTeam ? (
                <>
                  <b>,</b>
                  <br /> or go to your osquery packs via the ‘Advanced’ button.{" "}
                </>
              ) : (
                <>
                  <b>.</b>
                </>
              )}
            </p>
            <div className={`${noScheduleClass}__-cta-buttons`}>
              <Button
                variant="brand"
                className={`${noScheduleClass}__schedule-button`}
                onClick={toggleScheduleEditorModal}
              >
                Schedule a query
              </Button>
              {isOnGlobalTeam && (
                <Button
                  variant="inverse"
                  onClick={handleAdvanced}
                  className={`${baseClass}__advanced-button`}
                >
                  Advanced
                </Button>
              )}
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
        if (onEditScheduledQueryClick) {
          onEditScheduledQueryClick(global_scheduled_query);
        }
        break;
      default:
        if (onRemoveScheduledQueryClick) {
          onRemoveScheduledQueryClick([global_scheduled_query.id]);
        }
        break;
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const loadingTableData = useSelector((state: IRootState) => {
    if (selectedTeamData?.id) {
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

  const loadingInheritedQueriesTableData = useSelector((state: IRootState) => {
    return state.entities.global_scheduled_queries.isLoading;
  });

  if (inheritedQueries) {
    const inheritedQueriesTableHeaders = generateInheritedQueriesTableHeaders();

    return (
      <div className={`${baseClass}`}>
        <TableContainer
          resultsTitle={"queries"}
          columns={inheritedQueriesTableHeaders}
          data={generateDataSet(allScheduledQueriesList, selectedTeamData?.id)}
          isLoading={loadingInheritedQueriesTableData}
          defaultSortHeader={"query"}
          defaultSortDirection={"desc"}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={false}
          disablePagination
          disableCount
          emptyComponent={NoScheduledQueries}
        />
      </div>
    );
  }

  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={generateDataSet(allScheduledQueriesList, selectedTeamData?.id)}
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
