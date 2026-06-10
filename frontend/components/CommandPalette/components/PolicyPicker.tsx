import React from "react";
import { Command } from "cmdk";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import {
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
  IPolicyStats,
} from "interfaces/policy";
import CriticalPolicyBadge from "components/CriticalPolicyBadge";
import PillBadge from "components/PillBadge";
import { PATCH_TOOLTIP_CONTENT } from "components/SoftwareInstallPolicyBadges/SoftwareInstallPolicyBadges";

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";
import getFleetSuffix from "./pickerCopy";

const baseClass = "command-palette";

const POLICY_SEARCH_LIMIT = 50;

interface IPolicyPickerProps {
  search: string;
  currentTeam?: ITeamSummary;
  /** Critical-policy badge is Premium-only (matches PoliciesTable). */
  isPremiumTier?: boolean;
  onSelect: (policyId: number) => void;
}

const PolicyPicker = ({
  search,
  currentTeam,
  isPremiumTier = false,
  onSelect,
}: IPolicyPickerProps): JSX.Element => {
  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;

  const fleetSuffix = getFleetSuffix(currentTeam);

  const { items: policies, isLoading, debouncedQuery } = usePickerSearch<
    ILoadAllPoliciesResponse | ILoadTeamPoliciesResponse,
    IPolicyStats
  >({
    search,
    queryKeyPrefix: ["commandPalettePolicies", teamId ?? "global"],
    queryFn: (q) => {
      if (teamId !== undefined) {
        return teamPoliciesAPI.loadAllNew({
          teamId,
          page: 0,
          perPage: POLICY_SEARCH_LIMIT,
          query: q || undefined,
          // Surface inherited global policies in team views, matching the
          // Policies page behavior.
          mergeInherited: true,
        });
      }
      return globalPoliciesAPI.loadAllNew({
        page: 0,
        perPage: POLICY_SEARCH_LIMIT,
        query: q || undefined,
      });
    },
    selectItems: (data) => data?.policies ?? [],
  });

  if (isLoading && policies.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for policies...</div>;
  }

  if (policies.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>
        {debouncedQuery
          ? `No policies match "${debouncedQuery}"${fleetSuffix}.`
          : `No policies found${fleetSuffix}.`}
      </div>
    );
  }

  // "Inherited" applies only when viewing a specific team and the policy
  // is a global one (team_id === null), matching PoliciesTableConfig.
  const isViewingSpecificTeam =
    !!currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID;

  return (
    <Command.Group className={`${baseClass}__group`}>
      {policies.map((policy) => {
        const showCriticalBadge = isPremiumTier && policy.critical;
        const showPatchBadge = policy.type === "patch";
        const showInheritedBadge =
          isViewingSpecificTeam && policy.team_id === null;

        return (
          <Command.Item
            key={`policy-${policy.id}`}
            value={`${RESULT_PREFIXES.policy}${policy.id}`}
            onSelect={() => onSelect(policy.id)}
            className={`${baseClass}__item`}
          >
            <div className={`${baseClass}__item-left`}>
              <span className={`${baseClass}__item-label`}>{policy.name}</span>
              {showCriticalBadge && <CriticalPolicyBadge />}
              {showPatchBadge && (
                <PillBadge tipContent={PATCH_TOOLTIP_CONTENT}>Patch</PillBadge>
              )}
              {showInheritedBadge && (
                <PillBadge tipContent="This policy runs on all hosts.">
                  Inherited
                </PillBadge>
              )}
            </div>
          </Command.Item>
        );
      })}
    </Command.Group>
  );
};

export default PolicyPicker;
