/**
 * Component when there is an error retrieving schedule set up in fleet
 */
import React from "react";
import { InjectedRouter } from "react-router";
import paths from "router/paths";

import {
  IScheduledQuery,
  IEditScheduledQuery,
} from "interfaces/scheduled_query";
import { ITeam } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import {
  generateInheritedQueriesTableHeaders,
  generateTableHeaders,
  generateDataSet,
} from "./ScheduleTableConfig";

const baseClass = "schedule-table";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};
interface IScheduleTableProps {
  router: InjectedRouter; // v3
  onRemoveScheduledQueryClick?: (selectedIds: number[]) => void;
  onEditScheduledQueryClick?: (selectedQuery: IEditScheduledQuery) => void;
  onShowQueryClick?: (selectedQuery: IEditScheduledQuery) => void;
  allScheduledQueriesList: IScheduledQuery[];
  toggleScheduleEditorModal?: () => void;
  inheritedQueries?: boolean;
  isOnGlobalTeam: boolean;
  selectedTeamData: ITeam | undefined;
  loadingInheritedQueriesTableData: boolean;
  loadingTeamQueriesTableData: boolean;
}

const ScheduleTable = ({
  router,
  onRemoveScheduledQueryClick,
  onEditScheduledQueryClick,
  onShowQueryClick,
  allScheduledQueriesList,
  toggleScheduleEditorModal,
  inheritedQueries,
  isOnGlobalTeam,
  selectedTeamData,
  loadingInheritedQueriesTableData,
  loadingTeamQueriesTableData,
}: IScheduleTableProps): JSX.Element => {
  const { MANAGE_PACKS, MANAGE_HOSTS } = paths;

  const handleAdvanced = () => router.push(MANAGE_PACKS);

  const emptyState = () => {
    const emptySchedule: IEmptyTableProps = {
      iconName: "empty-schedule",
      header: (
        <>
          Schedule queries to run at regular intervals on{" "}
          <a href={MANAGE_HOSTS}>all your hosts</a>
        </>
      ),
      additionalInfo: (
        <>
          Want to learn more?&nbsp;
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query"
            text="Read about scheduling a query"
            newTab
          />
        </>
      ),
      primaryButton: (
        <Button
          variant="brand"
          className={`${baseClass}__schedule-button`}
          onClick={toggleScheduleEditorModal}
        >
          Schedule a query
        </Button>
      ),
    };

    if (selectedTeamData) {
      emptySchedule.header = (
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
      );
    }

    /* NOTE: Product decision to remove packs from UI
    if (isOnGlobalTeam) {
      emptySchedule.info = (
        <>Or go to your osquery packs via the ‘Advanced’ button. </>
      );
      emptySchedule.secondaryButton = (
        <Button
          variant="inverse"
          onClick={handleAdvanced}
          className={`${baseClass}__advanced-button`}
        >
          Advanced
        </Button>
      );
    }
    */
    return emptySchedule;
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
      case "showQuery":
        if (onShowQueryClick) {
          onShowQueryClick(scheduledQuery);
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
          emptyComponent={() =>
            EmptyTable({
              iconName: emptyState().iconName,
              header: emptyState().header,
              info: emptyState().info,
              additionalInfo: emptyState().additionalInfo,
              primaryButton: emptyState().primaryButton,
              secondaryButton: emptyState().secondaryButton,
            })
          }
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
        primarySelectAction={{
          name: "remove scheduled query",
          buttonText: "Remove",
          iconSvg: "ex",
          variant: "text-icon",
          onActionButtonClick: onRemoveScheduledQueryClick,
        }}
        emptyComponent={() =>
          EmptyTable({
            iconName: emptyState().iconName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
            secondaryButton: emptyState().secondaryButton,
          })
        }
        isClientSidePagination
      />
    </div>
  );
};

export default ScheduleTable;
