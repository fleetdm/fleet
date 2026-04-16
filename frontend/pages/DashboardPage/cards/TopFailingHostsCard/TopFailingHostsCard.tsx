import React from "react";
import { useQuery } from "react-query";

import chartsAPI, {
  IHostFailingSummary,
  ITopNonCompliantHostsResponse,
} from "services/entities/charts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "top-failing-hosts-card";

// DEFAULT_LIMIT matches the backend's clamped default. Fewer rows keeps the
// card compact next to the most-ignored-policies card.
const DEFAULT_LIMIT = 10;

// displayName picks the most human-readable name available. `computer_name` is
// the user-facing label on macOS (e.g. "Jane's MacBook"); `hostname` is the
// DNS/osquery identifier and is the fallback when computer_name is empty.
const displayName = (host: IHostFailingSummary): string => {
  if (host.computer_name && host.computer_name !== "") return host.computer_name;
  return host.hostname;
};

const renderRow = (host: IHostFailingSummary): JSX.Element => {
  let scope = "No fleet";
  if (host.team_id !== null) {
    scope = host.team_name || `Fleet #${host.team_id}`;
  }
  return (
    <tr key={host.host_id} className={`${baseClass}__row`}>
      <td className={`${baseClass}__name`}>{displayName(host)}</td>
      <td className={`${baseClass}__scope`}>{scope}</td>
      <td className={`${baseClass}__numeric`}>{host.failing_policy_count}</td>
    </tr>
  );
};

const TopFailingHostsCard = (): JSX.Element => {
  const { data, isFetching, error } = useQuery<
    ITopNonCompliantHostsResponse,
    Error
  >(
    ["top-non-compliant-hosts", DEFAULT_LIMIT],
    () => chartsAPI.getTopNonCompliantHosts(DEFAULT_LIMIT),
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
    const hosts = data?.hosts ?? [];
    if (!hosts.length) {
      return (
        <div className={`${baseClass}__no-data`}>
          No non-compliant hosts today.
        </div>
      );
    }
    return (
      <table className={`${baseClass}__table`}>
        <thead>
          <tr>
            <th className={`${baseClass}__name`}>Host</th>
            <th className={`${baseClass}__scope`}>Fleet</th>
            <th className={`${baseClass}__numeric`}>Failing policies</th>
          </tr>
        </thead>
        <tbody>{hosts.map(renderRow)}</tbody>
      </table>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <h2 className={`${baseClass}__title`}>Most non-compliant hosts</h2>
      </div>
      <div className={`${baseClass}__body`}>{renderBody()}</div>
    </div>
  );
};

export default TopFailingHostsCard;
