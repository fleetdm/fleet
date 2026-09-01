import React, { useEffect } from "react";
import { Command } from "cmdk";

import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";
import HighlightedLabel from "./HighlightedLabel";
import UprightEmoji from "./UprightEmoji";

const baseClass = "command-palette";

const HOST_SEARCH_LIMIT = 50;

interface IHostPickerProps {
  search: string;
  /** When true, render the host's team in a third column. Caller is
   *  responsible for enforcing premium-tier + non-Primo (single-fleet
   *  installs have nothing meaningful to show in the team column). */
  showTeamColumn?: boolean;
  onSelect: (hostId: number) => void;
  /** Fires when the results list identity changes, with the cmdk value of
   *  the first item (or null when empty). The parent uses it to point
   *  cmdk's highlight at the first row after each fetch — cmdk's own
   *  auto-select doesn't re-run when items reference-change. */
  onResultsChange?: (firstItemValue: string | null) => void;
}

const HostPicker = ({
  search,
  showTeamColumn = false,
  onSelect,
  onResultsChange,
}: IHostPickerProps): JSX.Element => {
  // No team scoping — the picker is a global navigator. On select, the
  // parent navigates to /hosts/:id/details without fleet_id; the host
  // details page reads the host's team from the host record itself, so
  // the user's current team context is preserved (matches the
  // ManageHostsPage.handleRowSelect pattern).
  const { items: hosts, isLoading, debouncedQuery } = usePickerSearch<
    ILoadHostsResponse,
    ILoadHostsResponse["hosts"][number]
  >({
    search,
    queryKeyPrefix: ["commandPaletteHosts"],
    queryFn: (q) =>
      hostsAPI.loadHosts({
        page: 0,
        perPage: HOST_SEARCH_LIMIT,
        globalFilter: q || undefined,
        sortBy: [{ key: "display_name", direction: "asc" }],
      }),
    selectItems: (data) => data?.hosts ?? [],
  });

  const firstItemValue =
    hosts.length > 0 ? `${RESULT_PREFIXES.host}${hosts[0].id}` : null;
  // Key the effect off a stable signature of the full list, not just the
  // first id. If results change but the first id is the same (user types
  // more, list shrinks/reorders), a previously-arrow'd row further down
  // may be gone — we still need to snap the highlight back to the first
  // row so the controlled cmdk value stays valid.
  const itemsSignature = hosts.map((h) => h.id).join(",");
  useEffect(() => {
    onResultsChange?.(firstItemValue);
  }, [itemsSignature, firstItemValue, onResultsChange]);

  if (isLoading && hosts.length === 0) {
    return <div className={`${baseClass}__empty`}>Looking for hosts...</div>;
  }

  if (hosts.length === 0) {
    return (
      <div className={`${baseClass}__empty`}>
        {debouncedQuery
          ? `No hosts match "${debouncedQuery}".`
          : "No hosts found."}
      </div>
    );
  }

  return (
    <Command.Group className={`${baseClass}__group`}>
      {hosts.map((host) => {
        const label = host.display_name || host.hostname || `Host ${host.id}`;
        const dotClass = `${baseClass}__host-status-dot ${baseClass}__host-status-dot--${host.status}`;
        return (
          <Command.Item
            key={`host-${host.id}`}
            value={`${RESULT_PREFIXES.host}${host.id}`}
            onSelect={() => onSelect(host.id)}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__host-name`}>
              <span
                className={dotClass}
                aria-label={`status: ${host.status}`}
              />
              <span className={`${baseClass}__item-label`}>
                {/* debouncedQuery, not live search — stays in sync
                    with the debounced row list. */}
                <HighlightedLabel text={label} query={debouncedQuery} />
              </span>
            </span>
            {showTeamColumn && (
              <span className={`${baseClass}__host-team`}>
                <UprightEmoji text={host.team_name || "Unassigned"} />
              </span>
            )}
          </Command.Item>
        );
      })}
    </Command.Group>
  );
};

export default HostPicker;
