import React, { useContext } from "react";
import { AppContext } from "context/app";
import { noop } from "lodash";
import paths from "router/paths";

import { IPolicyStats } from "interfaces/policy";
import { ITeamSummary } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";

const baseClass = "policies-table";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};

interface IPoliciesTableProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onAddPolicyClick?: () => void;
  onDeletePolicyClick: (selectedTableIds: number[]) => void;
  canAddOrDeletePolicy?: boolean;
  tableType?: string;
  currentTeam: ITeamSummary | undefined;
  currentAutomatedPolicies?: number[];
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
}: IPoliciesTableProps): JSX.Element => {
  const { MANAGE_HOSTS } = paths;

  const { config } = useContext(AppContext);

  const emptyState = () => {
    const emptyPolicies: IEmptyTableProps = {
      iconName: "empty-policies",
      header: (
        <>
          Ask yes or no questions about{" "}
          <a href={MANAGE_HOSTS}>all your hosts</a>
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
              MANAGE_HOSTS + TAGGED_TEMPLATES.hostsByTeamRoute(currentTeam.id)
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

    return emptyPolicies;
  };

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
          resultsTitle={"policies"}
          columns={generateTableHeaders({
            selectedTeamId: currentTeam?.id,
            canAddOrDeletePolicy,
            tableType,
          })}
          data={generateDataSet(
            policiesList,
            currentAutomatedPolicies,
            config?.update_interval.osquery_policy
          )}
          isLoading={isLoading}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          manualSortBy
          showMarkAllPages={false}
          isAllPagesSelected={false}
          onPrimarySelectActionClick={onDeletePolicyClick}
          primarySelectActionButtonVariant="text-icon"
          primarySelectActionButtonIcon="delete"
          primarySelectActionButtonText={"Delete"}
          emptyComponent={() =>
            EmptyTable({
              iconName: emptyState().iconName,
              header: emptyState().header,
              info: emptyState().info,
              additionalInfo: emptyState().additionalInfo,
              primaryButton: emptyState().primaryButton,
            })
          }
          onQueryChange={noop}
          disableCount={tableType === "inheritedPolicies"}
          isClientSidePagination
        />
      )}
    </div>
  );
};

export default PoliciesTable;
