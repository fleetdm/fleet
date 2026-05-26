import React, { useEffect, useState } from "react";
import { Command } from "cmdk";
import { useQuery } from "react-query";

import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";
import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import {
  ILoadAllPoliciesResponse,
  ILoadTeamPoliciesResponse,
} from "interfaces/policy";

const baseClass = "command-palette";

const POLICY_SEARCH_LIMIT = 50;
const POLICY_SEARCH_DEBOUNCE_MS = 200;

interface IPolicyPickerProps {
  search: string;
  currentTeam?: ITeamSummary;
  onSelect: (policyId: number) => void;
}

const PolicyPicker = ({
  search,
  currentTeam,
  onSelect,
}: IPolicyPickerProps): JSX.Element => {
  const [debouncedQuery, setDebouncedQuery] = useState(search.trim());
  useEffect(() => {
    const id = window.setTimeout(() => {
      setDebouncedQuery(search.trim());
    }, POLICY_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [search]);

  const teamId =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.id
      : undefined;
  const fleetLabel =
    currentTeam && currentTeam.id !== APP_CONTEXT_ALL_TEAMS_ID
      ? currentTeam.name
      : "All fleets";

  // Policies live in two APIs: global (no team / All fleets) and team
  // (specific team or Unassigned). Branch on teamId.
  const { data, isLoading } = useQuery<
    ILoadAllPoliciesResponse | ILoadTeamPoliciesResponse,
    Error
  >(
    ["commandPalettePolicies", teamId ?? "global", debouncedQuery],
    () => {
      if (teamId !== undefined) {
        return teamPoliciesAPI.loadAllNew({
          teamId,
          page: 0,
          perPage: POLICY_SEARCH_LIMIT,
          query: debouncedQuery || undefined,
        });
      }
      return globalPoliciesAPI.loadAllNew({
        page: 0,
        perPage: POLICY_SEARCH_LIMIT,
        query: debouncedQuery || undefined,
      });
    },
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  const policies = data?.policies ?? [];

  if (isLoading && policies.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>Looking for policies...</div>
    );
  }

  if (policies.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>
        {debouncedQuery
          ? `No policies match "${debouncedQuery}" in ${fleetLabel}.`
          : `No policies found in ${fleetLabel}.`}
      </div>
    );
  }

  return (
    <Command.Group className={`${baseClass}__group`}>
      {policies.map((policy) => (
        <Command.Item
          key={`policy-${policy.id}`}
          value={`POLICY_RESULT ${policy.id}`}
          onSelect={() => onSelect(policy.id)}
          className={`${baseClass}__item`}
        >
          <span className={`${baseClass}__item-label`}>{policy.name}</span>
        </Command.Item>
      ))}
    </Command.Group>
  );
};

export default PolicyPicker;
