import React from "react";
import { Command } from "cmdk";

import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";

import usePickerSearch from "./usePickerSearch";
import { RESULT_PREFIXES } from "./constants";

const baseClass = "command-palette";

const HOST_SEARCH_LIMIT = 50;

interface IHostPickerProps {
  search: string;
  onSelect: (hostId: number) => void;
}

const HostPicker = ({ search, onSelect }: IHostPickerProps): JSX.Element => {
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
        return (
          <Command.Item
            key={`host-${host.id}`}
            value={`${RESULT_PREFIXES.host}${host.id}`}
            onSelect={() => onSelect(host.id)}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__item-label`}>{label}</span>
          </Command.Item>
        );
      })}
    </Command.Group>
  );
};

export default HostPicker;
