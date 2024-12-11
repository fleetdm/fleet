import React, { useContext } from "react";
import { AppContext } from "context/app";

import { IPolicyStats } from "interfaces/policy";
import { ITeamSummary } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";

const baseClass = "policies-table";

const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";

interface IPoliciesTableProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onAddPolicyClick?: () => void;
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
  resetPageIndex: boolean;
}

const PoliciesTable = ({
  policiesList,
  isLoading,
  onAddPolicyClick,
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
  resetPageIndex,
}: IPoliciesTableProps): JSX.Element => {
  const { config } = useContext(AppContext);

  const emptyState = () => {
    const emptyPolicies: IEmptyTableProps = {
      graphicName: "empty-policies",
      header: "You don't have any policies",
      info:
        "Add policies to detect device health issues and trigger automations.",
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
      emptyPolicies.header = "No matching policies";
      emptyPolicies.info = "No policies match the current filters.";
    }

    return emptyPolicies;
  };

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
          policiesList,
          isPremiumTier
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
        renderCount={renderPoliciesCount}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable={searchable}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default PoliciesTable;
