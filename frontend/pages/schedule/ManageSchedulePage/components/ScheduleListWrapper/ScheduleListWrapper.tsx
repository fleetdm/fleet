/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React from "react";
import { InjectedRouter } from "react-router";
import paths from "router/paths";

import Button from "components/buttons/Button";
import {
  IScheduledQuery,
  IEditScheduledQuery,
} from "interfaces/scheduled_query";

import { ITeam } from "interfaces/team";

import TableContainer from "components/TableContainer";
import {
  generateInheritedQueriesTableHeaders,
  generateTableHeaders,
  generateDataSet,
} from "./ScheduleTableConfig";
import scheduleSvg from "../../../../../../assets/images/no-schedule-322x138@2x.png";

const baseClass = "schedule-list-wrapper";
const noScheduleClass = "no-schedule";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};
interface IScheduleListWrapperProps {
  router: InjectedRouter; // v3
  onRemoveScheduledQueryClick?: (selectIds: number[]) => void;
  onEditScheduledQueryClick?: (selectedQuery: IEditScheduledQuery) => void;
  allScheduledQueriesList: IScheduledQuery[];
  toggleScheduleEditorModal?: () => void;
  inheritedQueries?: boolean;
  isOnGlobalTeam: boolean;
  selectedTeamData: ITeam | undefined;
  loadingInheritedQueriesTableData: boolean;
  loadingTeamQueriesTableData: boolean;
}

const ScheduleListWrapper = ({
  router,
  onRemoveScheduledQueryClick,
  allScheduledQueriesList,
  toggleScheduleEditorModal,
  onEditScheduledQueryClick,
  inheritedQueries,
  isOnGlobalTeam,
  selectedTeamData,
  loadingInheritedQueriesTableData,
  loadingTeamQueriesTableData,
}: IScheduleListWrapperProps): JSX.Element => {
  const { MANAGE_PACKS, MANAGE_HOSTS } = paths;

  const handleAdvanced = () => router.push(MANAGE_PACKS);

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
                        TAGGED_TEMPLATES.hostsByTeamRoute(selectedTeamData.id)
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
            <div className={`${noScheduleClass}__cta-buttons`}>
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
    scheduledQuery: IEditScheduledQuery
  ): void => {
    switch (action) {
      case "edit":
        if (onEditScheduledQueryClick) {
          onEditScheduledQueryClick(scheduledQuery);
        }
        break;
      default:
        if (onRemoveScheduledQueryClick) {
          onRemoveScheduledQueryClick([scheduledQuery.id]);
        }
        break;
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const loadingTableData = selectedTeamData?.id
    ? loadingTeamQueriesTableData
    : loadingInheritedQueriesTableData;

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
