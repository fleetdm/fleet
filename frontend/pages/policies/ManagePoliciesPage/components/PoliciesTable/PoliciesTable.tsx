import React, { useContext } from "react";
import { AppContext } from "context/app";

import { IPolicyStats } from "interfaces/policy";
import { ITeamSummary, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";
import {
  DEFAULT_SORT_COLUMN,
  DEFAULT_SORT_DIRECTION,
  DEFAULT_PAGE_SIZE,
} from "../../ManagePoliciesPage";

// isLastPage is removable if/when API is updated to include meta.has_next_results
const isLastPage = (count: number, pageSize: number, page: number) => {
  return count <= pageSize * (page + 1);
};

const baseClass = "policies-table";

interface IPoliciesTableProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onDeletePolicyClick: (selectedTableIds: number[]) => void;
  canAddOrDeletePolicy?: boolean;
  hasPoliciesToDelete?: boolean;
  currentTeam: ITeamSummary | undefined;
  currentAutomatedPolicies?: number[];
  isPremiumTier?: boolean;
  renderPoliciesCount: () => JSX.Element | null;
  onQueryChange: (newTableQuery: ITableQueryData) => void;
  searchQuery: string;
  sortHeader?: "name" | "failing_host_count";
  sortDirection?: "asc" | "desc";
  page: number;
  count: number;
}

const PoliciesTable = ({
  policiesList,
  isLoading,
  onDeletePolicyClick,
  canAddOrDeletePolicy,
  hasPoliciesToDelete,
  currentTeam,
  currentAutomatedPolicies,
  isPremiumTier,
  onQueryChange,
  renderPoliciesCount,
  searchQuery,
  sortHeader,
  sortDirection,
  page,
  count,
}: IPoliciesTableProps): JSX.Element => {
  const { config } = useContext(AppContext);

  const emptyState: IEmptyTableProps = {
    graphicName: "empty-policies",
    header: "You don't have any policies",
    info:
      "Add policies to detect device health issues and trigger automations.",
  };

  if (isPremiumTier) {
    if (
      currentTeam?.id === null ||
      currentTeam?.id === APP_CONTEXT_ALL_TEAMS_ID
    ) {
      emptyState.header += " that apply to all teams";
    } else {
      emptyState.header += " that apply to this team";
    }
  }

  if (!canAddOrDeletePolicy) {
    emptyState.info = "";
  }

  if (searchQuery) {
    delete emptyState.graphicName;
    delete emptyState.primaryButton;
    emptyState.header = "No matching policies";
    emptyState.info = "No policies match the current filters.";
  }

  const searchable = !(policiesList?.length === 0 && searchQuery === "");

  const hasPermissionAndPoliciesToDelete =
    canAddOrDeletePolicy && hasPoliciesToDelete;

  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="policies"
        columnConfigs={generateTableHeaders(
          {
            selectedTeamId: currentTeam?.id,
            hasPermissionAndPoliciesToDelete,
          },
          isPremiumTier
        )}
        data={generateDataSet(
          policiesList,
          currentAutomatedPolicies,
          config?.update_interval.osquery_policy
        )}
        isLoading={isLoading}
        defaultSortHeader={sortHeader || DEFAULT_SORT_COLUMN}
        defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
        defaultSearchQuery={searchQuery}
        pageIndex={page}
        disableNextPage={isLastPage(count, DEFAULT_PAGE_SIZE, page)}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        primarySelectAction={{
          name: "delete policy",
          buttonText: "Delete",
          iconSvg: "trash",
          variant: "text-icon",
          onClick: onDeletePolicyClick,
        }}
        emptyComponent={() =>
          EmptyTable({
            graphicName: emptyState.graphicName,
            header: emptyState.header,
            info: emptyState.info,
            additionalInfo: emptyState.additionalInfo,
            primaryButton: emptyState.primaryButton,
          })
        }
        renderCount={renderPoliciesCount}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable={searchable}
      />
    </div>
  );
};

export default PoliciesTable;
