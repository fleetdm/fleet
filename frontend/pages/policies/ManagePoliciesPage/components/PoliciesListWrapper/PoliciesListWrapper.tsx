import React from "react";
import { noop } from "lodash";
import paths from "router/paths";

import { IPolicyStats } from "interfaces/policy";
import { ITeam } from "interfaces/team";
import TableContainer from "components/TableContainer";
import { generateTableHeaders, generateDataSet } from "./PoliciesTableConfig";
// @ts-ignore
import policySvg from "../../../../../../assets/images/no-policy-323x138@2x.png";

const baseClass = "policies-list-wrapper";
const noPoliciesClass = "no-policies";

const TAGGED_TEMPLATES = {
  hostsByTeamRoute: (teamId: number | undefined | null) => {
    return `${teamId ? `/?team_id=${teamId}` : ""}`;
  },
};

interface IPoliciesListWrapperProps {
  policiesList: IPolicyStats[];
  isLoading: boolean;
  onRemovePoliciesClick: (selectedTableIds: number[]) => void;
  resultsTitle?: string;
  canAddOrRemovePolicy?: boolean;
  tableType?: string;
  selectedTeamData: ITeam | undefined;
  toggleAddPolicyModal?: () => void;
}

const PoliciesListWrapper = ({
  policiesList,
  isLoading,
  onRemovePoliciesClick,
  resultsTitle,
  canAddOrRemovePolicy,
  tableType,
  selectedTeamData,
  toggleAddPolicyModal,
}: IPoliciesListWrapperProps): JSX.Element => {
  const { MANAGE_HOSTS } = paths;

  const NoPolicies = () => {
    return (
      <div
        className={`${noPoliciesClass} ${
          selectedTeamData?.id && "no-team-policy"
        }`}
      >
        <div className={`${noPoliciesClass}__inner`}>
          <img src={policySvg} alt="No Policies" />
          <div className={`${noPoliciesClass}__inner-text`}>
            <p>
              <b>
                {selectedTeamData ? (
                  <>
                    Ask yes or no questions about hosts assigned to{" "}
                    <a
                      href={
                        MANAGE_HOSTS +
                        TAGGED_TEMPLATES.hostsByTeamRoute(selectedTeamData.id)
                      }
                    >
                      {selectedTeamData.name}
                    </a>
                    .
                  </>
                ) : (
                  <>
                    Ask yes or no questions about{" "}
                    <a href={MANAGE_HOSTS}>all your hosts</a>.
                  </>
                )}
              </b>
            </p>
            <div className={`${noPoliciesClass}__bullet-text`}>
              <p>
                - Verify whether or not your hosts have security features turned
                on.
                <br />- Track your efforts to keep installed software up to date
                on your hosts.
                <br />- Provide owners with a list of hosts that still need
                changes.
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div
      className={`${baseClass} ${
        canAddOrRemovePolicy ? "" : "hide-selection-column"
      }`}
    >
      <TableContainer
        resultsTitle={resultsTitle || "policies"}
        columns={generateTableHeaders({
          selectedTeamId: selectedTeamData?.id,
          showSelectionColumn: canAddOrRemovePolicy,
          tableType,
        })}
        data={generateDataSet(policiesList)}
        isLoading={isLoading}
        defaultSortHeader={"name"}
        defaultSortDirection={"asc"}
        manualSortBy
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disablePagination
        onPrimarySelectActionClick={onRemovePoliciesClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={NoPolicies}
        onQueryChange={noop}
        disableCount={tableType === "inheritedPolicies"}
      />
    </div>
  );
};

export default PoliciesListWrapper;
