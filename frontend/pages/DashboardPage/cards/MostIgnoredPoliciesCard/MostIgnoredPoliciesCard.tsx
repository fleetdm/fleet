import React from "react";
import { useQuery } from "react-query";

import chartsAPI, {
  IMostIgnoredPolicy,
  IMostIgnoredPoliciesResponse,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "most-ignored-policies-card";

// DEFAULT_LIMIT is the number of policies shown. Backend default is unlimited,
// but the card is a dashboard widget — a fixed top-N keeps it glanceable.
const DEFAULT_LIMIT = 10;

const renderRow = (policy: IMostIgnoredPolicy): JSX.Element => {
  // Prefer the team_name from the API; fall back to "Fleet #N" if we know the
  // ID but somehow missed the name lookup, or "Global" for unscoped policies.
  let scope = "Global";
  if (policy.team_id !== null) {
    scope = policy.team_name || `Fleet #${policy.team_id}`;
  }
  return (
    <tr key={policy.policy_id} className={`${baseClass}__row`}>
      <td className={`${baseClass}__name`}>{policy.name}</td>
      <td className={`${baseClass}__scope`}>{scope}</td>
      <td className={`${baseClass}__numeric`}>{policy.failing_host_count}</td>
    </tr>
  );
};

const MostIgnoredPoliciesCard = (): JSX.Element => {
  const { data, isFetching, error } = useQuery<
    IMostIgnoredPoliciesResponse,
    Error
  >(
    ["most-ignored-policies", DEFAULT_LIMIT],
    () => chartsAPI.getMostIgnoredPolicies(DEFAULT_LIMIT),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 300000, // 5 minutes
    }
  );

  const renderBody = (): JSX.Element => {
    if (isFetching) {
      return <Spinner includeContainer={false} verticalPadding="small" />;
    }
    if (error) {
      return <DataError />;
    }
    if (!data?.policies?.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          No policy compliance data yet.
        </div>
      );
    }
    // Filter out fully-passing policies — the list is meant to draw the eye to
    // the things people are actually ignoring. We still *store* them so "count
    // of policies tracked" is derivable elsewhere.
    const rows = data.policies.filter((p) => p.failing_host_count > 0);
    if (!rows.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          Nothing is being ignored. Everyone is passing every tracked policy.
        </div>
      );
    }
    return (
      <table className={`${baseClass}__table`}>
        <thead>
          <tr>
            <th className={`${baseClass}__name`}>Policy</th>
            <th className={`${baseClass}__scope`}>Scope</th>
            <th className={`${baseClass}__numeric`}>Failing hosts</th>
          </tr>
        </thead>
        <tbody>{rows.map(renderRow)}</tbody>
      </table>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <h2 className={`${baseClass}__title`}>Most-ignored policies</h2>
      </div>
      <div className={`${baseClass}__body`}>{renderBody()}</div>
    </div>
  );
};

export default MostIgnoredPoliciesCard;
