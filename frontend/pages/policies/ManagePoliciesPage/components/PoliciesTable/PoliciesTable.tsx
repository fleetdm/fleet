import React, { useCallback, useContext } from "react";
import { InjectedRouter } from "react-router";
import { SingleValue } from "react-select-5";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import { IPolicyStats, OtherAutomationType } from "interfaces/policy";
import { ITeamSummary, APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import { IEmptyStateProps } from "interfaces/empty_state";
import { SelectedPlatform } from "interfaces/platform";
import { getNextLocationPath } from "utilities/helpers";
import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import EmptyState from "components/EmptyState";
import { AutomationType } from "services/entities/team_policies";
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

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
  },
  {
    disabled: false,
    label: "ChromeOS",
    value: "chrome",
  },
];

interface IPoliciesTableProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onDeletePoliciesClick: (selectedTableIds: number[]) => void;
  onAddPolicyClick: () => void;
  canAddOrDeletePolicies?: boolean;
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
  customControl?: () => JSX.Element | null;
  isFiltered?: boolean;
  router: InjectedRouter;
  queryParams?: {
    fleet_id?: string;
    query?: string;
    order_key?: string;
    order_direction?: "asc" | "desc";
    page?: string;
    automation_type?: AutomationType;
    platform?: string;
  };
  platform?: SelectedPlatform;
  otherAutomationType?: OtherAutomationType;
  onOpenManageAutomationsModal?: (policy: IPolicyStats) => void;
}

const PoliciesTable = ({
  policiesList,
  isLoading,
  onDeletePoliciesClick,
  onAddPolicyClick,
  canAddOrDeletePolicies,
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
  customControl,
  isFiltered,
  router,
  queryParams,
  platform = "all",
  otherAutomationType,
  onOpenManageAutomationsModal,
}: IPoliciesTableProps): JSX.Element => {
  const { config } = useContext(AppContext);

  const handlePlatformFilterDropdownChange = useCallback(
    (selectedTargetedPlatform: SingleValue<CustomOptionType>) => {
      router.push(
        getNextLocationPath({
          pathPrefix: PATHS.MANAGE_POLICIES,
          queryParams: {
            ...queryParams,
            page: 0,
            platform:
              selectedTargetedPlatform?.value === "all"
                ? undefined
                : selectedTargetedPlatform?.value,
          },
        })
      );
    },
    [queryParams, router]
  );

  const renderPlatformDropdown = useCallback(() => {
    return (
      <DropdownWrapper
        name="platform-dropdown"
        value={platform}
        className={`${baseClass}__platform-dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        onChange={handlePlatformFilterDropdownChange}
        variant="table-filter"
        iconName="filter-alt"
      />
    );
  }, [platform, handlePlatformFilterDropdownChange]);

  const isAllFleets =
    isPremiumTier &&
    (currentTeam?.id === null || currentTeam?.id === APP_CONTEXT_ALL_TEAMS_ID);

  let emptyHeader = "No policies yet";
  // Primo mode uses a generic empty state header
  if (isPremiumTier && !config?.partnerships?.enable_primo) {
    emptyHeader = isAllFleets
      ? "No policies apply to all fleets"
      : "No policies for this fleet";
  }

  const emptyState: IEmptyStateProps = {
    header: emptyHeader,
    info:
      "Policies are queries that return a pass or fail result. Failures trigger fixes or prompt end users to solve them on their own.",
    primaryButton: canAddOrDeletePolicies ? (
      <Button onClick={onAddPolicyClick} type="button">
        Add policy
      </Button>
    ) : undefined,
  };

  if (searchQuery || isFiltered) {
    delete emptyState.primaryButton;
    emptyState.header = "No matching policies";
    emptyState.info = "No policies match the current filters.";
  }

  const isTrulyEmpty =
    policiesList?.length === 0 && searchQuery === "" && !isFiltered;

  const combinedCustomControl = () => {
    return (
      <div className={`${baseClass}__filter-dropdowns`}>
        {customControl?.()}
        {renderPlatformDropdown()}
      </div>
    );
  };

  const isPrimoMode = config?.partnerships?.enable_primo || false;
  const viewingTeamPolicies =
    currentTeam?.id !== undefined &&
    currentTeam?.id !== null &&
    currentTeam?.id !== APP_CONTEXT_ALL_TEAMS_ID;

  // Hide the selection column if the current page has no selectable rows
  // (e.g., all rows are inherited policies that can't be selected)
  const pageHasSelectableRows =
    !viewingTeamPolicies ||
    isPrimoMode ||
    policiesList.some((p) => p.team_id !== null);

  const hasPermissionAndPoliciesToDelete =
    canAddOrDeletePolicies && hasPoliciesToDelete && pageHasSelectableRows;

  return (
    <div className={baseClass}>
      <TableContainer
        resultsTitle="policies"
        columnConfigs={generateTableHeaders(
          {
            selectedTeamId: currentTeam?.id,
            hasPermissionAndPoliciesToDelete,
            otherAutomationType,
            onOpenManageAutomationsModal,
          },
          isPremiumTier,
          config?.partnerships?.enable_primo
        )}
        data={generateDataSet(
          policiesList,
          currentAutomatedPolicies,
          config?.update_interval?.osquery_policy
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
          variant: "secondary",
          onClick: onDeletePoliciesClick,
        }}
        emptyComponent={() => (
          <EmptyState
            header={emptyState.header}
            info={emptyState.info}
            additionalInfo={emptyState.additionalInfo}
            primaryButton={emptyState.primaryButton}
          />
        )}
        renderCount={renderPoliciesCount}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable
        disableSearch={isTrulyEmpty}
        customControl={combinedCustomControl}
        selectedDropdownFilter={platform}
      />
    </div>
  );
};

export default PoliciesTable;
