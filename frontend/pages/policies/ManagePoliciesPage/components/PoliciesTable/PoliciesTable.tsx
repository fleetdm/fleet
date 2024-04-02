import React, { useContext } from "react";
import { AppContext } from "context/app";
import PATHS from "router/paths";

import { IPolicyStats } from "interfaces/policy";
import { ITeamSummary } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
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
  // onClientSidePaginationChange?: (pageIndex: number) => void;
  renderPoliciesCount: any; // TODO: typing
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
  // onClientSidePaginationChange,
  renderPoliciesCount,
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
      graphicName: "empty-policies",
      header: <>You don&apos;t have any policies</>,
      info: (
        <>
          Add policies to detect device health issues and trigger automations.
        </>
      ),
    };
    if (canAddOrDeletePolicy) {
      emptyPolicies.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__select-policy-button`}
          onClick={onAddPolicyClick}
        >
          Add policy
        </Button>
      );
    }
    if (searchQuery) {
      delete emptyPolicies.graphicName;
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
      <TableContainer
        resultsTitle="policies"
        columnConfigs={generateTableHeaders(
          {
            selectedTeamId: currentTeam?.id,
            canAddOrDeletePolicy,
            tableType,
          },
          policiesList,
          isPremiumTier,
          isSandboxMode
        )}
        data={generateDataSet(
          policiesList,
          currentAutomatedPolicies,
          config?.update_interval.osquery_policy
        )}
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
            graphicName: emptyState().graphicName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })
        }
        disableCount={tableType === "inheritedPolicies"}
        renderCount={renderPoliciesCount}
        onQueryChange={onTableQueryChange}
        inputPlaceHolder="Search by name"
        searchable={searchable}
      />
    </div>
  );
};

export default PoliciesTable;
