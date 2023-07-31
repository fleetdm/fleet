import React, { useContext } from "react";
import { AppContext } from "context/app";
import PATHS from "router/paths";

import { IPolicyStats } from "interfaces/policy";
import { ITeamSummary } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";

const baseClass = "policies-table";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};

const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";

interface IPoliciesTableProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onAddPolicyClick?: () => void;
  onDeletePolicyClick: (selectedTableIds: number[]) => void;
  canAddOrDeletePolicy?: boolean;
  tableType?: "inheritedPolicies";
  currentTeam: ITeamSummary | undefined;
  currentAutomatedPolicies?: number[];
  isPremiumTier?: boolean;
  isSandboxMode?: boolean;
  onClientSidePaginationChange?: (pageIndex: number) => void;
  onQueryChange: (newTableQuery: ITableQueryData) => void;
  searchQuery: string;
  sortHeader?: "name" | "failing_host_count";
  sortDirection?: "asc" | "desc";
  page: number;
}

const PoliciesTable = ({
  policiesList,
  isLoading,
  onAddPolicyClick,
  onDeletePolicyClick,
  canAddOrDeletePolicy,
  tableType,
  currentTeam,
  currentAutomatedPolicies,
  isPremiumTier,
  isSandboxMode,
  onQueryChange,
  onClientSidePaginationChange,
  searchQuery,
  sortHeader,
  sortDirection,
  page,
}: IPoliciesTableProps): JSX.Element => {
  const { config } = useContext(AppContext);

  // Inherited table uses the same onQueryChange but require different URL params
  const onTableQueryChange = (newTableQuery: ITableQueryData) => {
    onQueryChange({
      ...newTableQuery,
      editingInheritedTable: tableType === "inheritedPolicies",
    });
  };

  const emptyState = () => {
    const emptyPolicies: IEmptyTableProps = {
      iconName: "empty-policies",
      header: (
        <>
          Ask yes or no questions about{" "}
          <a href={PATHS.MANAGE_HOSTS}>all your hosts</a>
        </>
      ),
      info: (
        <>
          - Verify whether or not your hosts have security features turned on.
          <br />- Track your efforts to keep installed software up to date on
          your hosts.
          <br />- Provide owners with a list of hosts that still need changes.
        </>
      ),
    };

    if (currentTeam) {
      emptyPolicies.header = (
        <>
          Ask yes or no questions about hosts assigned to{" "}
          <a
            href={
              PATHS.MANAGE_HOSTS +
              TAGGED_TEMPLATES.hostsByTeamRoute(currentTeam.id)
            }
          >
            {currentTeam.name}
          </a>
        </>
      );
    }
    if (canAddOrDeletePolicy) {
      emptyPolicies.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__select-policy-button`}
          onClick={onAddPolicyClick}
        >
          Add a policy
        </Button>
      );
    }
    if (searchQuery) {
      delete emptyPolicies.iconName;
      delete emptyPolicies.primaryButton;
      emptyPolicies.header = "No policies match the current search criteria.";
      emptyPolicies.info =
        "Expecting to see policies? Try again in a few seconds as the system catches up.";
    }

    return emptyPolicies;
  };

  const searchable = !(policiesList?.length === 0 && searchQuery === "");

  return (
    <div
      className={`${baseClass} ${
        canAddOrDeletePolicy ? "" : "hide-selection-column"
      }`}
    >
      {isLoading ? (
        <Spinner />
      ) : (
        <TableContainer
          resultsTitle="policies"
          columns={generateTableHeaders(
            {
              selectedTeamId: currentTeam?.id,
              canAddOrDeletePolicy,
              tableType,
            },
            isPremiumTier,
            isSandboxMode
          )}
          data={generateDataSet(
            policiesList,
            currentAutomatedPolicies,
            config?.update_interval.osquery_policy
          )}
          filters={{ global: searchQuery }}
          isLoading={isLoading}
          defaultSortHeader={sortHeader || DEFAULT_SORT_HEADER}
          defaultSortDirection={sortDirection || DEFAULT_SORT_DIRECTION}
          defaultSearchQuery={searchQuery}
          defaultPageIndex={page}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          primarySelectAction={{
            name: "delete policy",
            buttonText: "Delete",
            iconSvg: "trash",
            variant: "text-icon",
            onActionButtonClick: onDeletePolicyClick,
          }}
          emptyComponent={() =>
            EmptyTable({
              iconName: emptyState().iconName,
              header: emptyState().header,
              info: emptyState().info,
              additionalInfo: emptyState().additionalInfo,
              primaryButton: emptyState().primaryButton,
            })
          }
          disableCount={tableType === "inheritedPolicies"}
          isClientSidePagination
          onClientSidePaginationChange={onClientSidePaginationChange}
          isClientSideFilter
          searchQueryColumn="name"
          onQueryChange={onTableQueryChange}
          inputPlaceHolder="Search by name"
          searchable={searchable}
        />
      )}
    </div>
  );
};

export default PoliciesTable;
