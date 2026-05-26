import React, { useEffect, useState } from "react";
import { Command } from "cmdk";
import { useQuery } from "react-query";

import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";

const baseClass = "command-palette";

const HOST_SEARCH_LIMIT = 50;
const HOST_SEARCH_DEBOUNCE_MS = 200;

interface IHostPickerProps {
  search: string;
  onSelect: (hostId: number) => void;
}

const HostPicker = ({ search, onSelect }: IHostPickerProps): JSX.Element => {
  // Debounce the raw search input so we don't fire a request per keystroke.
  const [debouncedQuery, setDebouncedQuery] = useState(search.trim());
  useEffect(() => {
    const id = window.setTimeout(() => {
      setDebouncedQuery(search.trim());
    }, HOST_SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(id);
  }, [search]);

  // No team scoping — the picker is a global navigator. The user is looking
  // for a specific host by name; the selected host's own team_id is used to
  // route to the correctly-scoped details URL on select.
  const { data, isLoading } = useQuery<ILoadHostsResponse, Error>(
    ["commandPaletteHosts", debouncedQuery],
    () =>
      hostsAPI.loadHosts({
        page: 0,
        perPage: HOST_SEARCH_LIMIT,
        globalFilter: debouncedQuery || undefined,
        sortBy: [{ key: "display_name", direction: "asc" }],
      }),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  const hosts = data?.hosts ?? [];

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
        // value prefixed with HOST_RESULT so cmdk's local filter passes it
        // through — the server already filtered by debouncedQuery.
        return (
          <Command.Item
            key={`host-${host.id}`}
            value={`HOST_RESULT ${host.id}`}
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
