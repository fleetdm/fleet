import React from "react";
import { waitFor } from "@testing-library/react";
import { Command } from "cmdk";
import { createCustomRenderer } from "test/test-utils";

import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";

import HostPicker from "./HostPicker";

// cmdk uses scrollIntoView which JSDOM doesn't implement
Element.prototype.scrollIntoView = jest.fn();

jest.mock("services/entities/hosts", () => ({
  __esModule: true,
  default: { loadHosts: jest.fn() },
}));

const mockedHosts = hostsAPI as jest.Mocked<typeof hostsAPI>;

const renderInClient = createCustomRenderer({ withBackendMock: true });
// Command.Item needs a Command root in context; the parent dialog supplies
// one in production, so wrap here for tests that actually render items.
const renderPicker = (
  ui: React.ReactElement
): ReturnType<typeof renderInClient> => renderInClient(<Command>{ui}</Command>);

// Minimal valid response — the picker only reads `hosts`, so we cast
// through `unknown` rather than constructing the unused munki/MDM
// aggregates. The typed local lets future renames of the field this
// test cares about surface here.
const emptyHostsResponse: ILoadHostsResponse = ({
  hosts: [],
} as unknown) as ILoadHostsResponse;

const hostsResponseWith = (
  ...hosts: Array<{
    id: number;
    display_name: string;
    status: string;
    team_id: number | null;
    team_name: string | null;
  }>
): ILoadHostsResponse =>
  (({
    hosts,
  } as unknown) as ILoadHostsResponse);

beforeEach(() => {
  mockedHosts.loadHosts.mockReset();
  mockedHosts.loadHosts.mockResolvedValue(emptyHostsResponse);
});

describe("HostPicker", () => {
  it("calls loadHosts WITHOUT teamId (global navigator, no scoping)", async () => {
    renderPicker(<HostPicker search="" onSelect={jest.fn()} />);

    await waitFor(() => {
      expect(mockedHosts.loadHosts).toHaveBeenCalled();
    });
    // Confirm the teamId key is absent entirely — `expect.anything()`
    // matches values but skips null/undefined, so it can't catch a
    // future regression that passes `teamId: undefined`.
    const callArgs = mockedHosts.loadHosts.mock.calls[0][0] as Record<
      string,
      unknown
    >;
    expect(Object.keys(callArgs)).not.toContain("teamId");
  });

  it("passes search as globalFilter and sorts by display_name asc", async () => {
    renderPicker(<HostPicker search="rachel" onSelect={jest.fn()} />);

    await waitFor(() => {
      expect(mockedHosts.loadHosts).toHaveBeenCalledWith(
        expect.objectContaining({
          globalFilter: "rachel",
          sortBy: [{ key: "display_name", direction: "asc" }],
        })
      );
    });
  });

  it("renders the no-search empty state when no hosts return", async () => {
    const { findByText } = renderPicker(
      <HostPicker search="" onSelect={jest.fn()} />
    );
    expect(await findByText(/No hosts found\./)).toBeInTheDocument();
  });

  it("renders a search-specific empty state when a debounced query returns nothing", async () => {
    const { findByText } = renderPicker(
      <HostPicker search="nonexistent" onSelect={jest.fn()} />
    );
    expect(
      await findByText(/No hosts match "nonexistent"\./)
    ).toBeInTheDocument();
  });

  describe("columns", () => {
    const hosts = hostsResponseWith({
      id: 1,
      display_name: "Rachel's MacBook",
      status: "online",
      team_id: 5,
      team_name: "Engineering",
    });

    // The shared QueryClient persists React Query's cache across tests
    // in this file. Earlier tests register an empty result under
    // queryKey ["commandPaletteHosts", ""], so subsequent renders with
    // the same search would read that cached emptiness and never hit
    // the mock. Each column test uses a unique search string to get a
    // fresh queryFn invocation. The mock ignores the query value, so
    // the same `hosts` is returned regardless.
    it("renders a status dot next to the host name (no text)", async () => {
      mockedHosts.loadHosts.mockResolvedValue(hosts);

      const { findByText, container } = renderPicker(
        <HostPicker search="col-dot-test" onSelect={jest.fn()} />
      );
      expect(await findByText("Rachel's MacBook")).toBeInTheDocument();

      // The dot is a presentational span; assert by class so the test
      // pins both the existence and the status-specific modifier.
      const dot = container.querySelector(
        ".command-palette__host-status-dot--online"
      );
      expect(dot).toBeInTheDocument();
      // No status text rendered alongside the dot.
      expect(container.textContent).not.toMatch(/Online/);
    });

    it("renders the host's team in the right-aligned column when showTeamColumn", async () => {
      mockedHosts.loadHosts.mockResolvedValue(hosts);

      const { findByText } = renderPicker(
        <HostPicker search="col-team-on" showTeamColumn onSelect={jest.fn()} />
      );
      expect(await findByText("Engineering")).toBeInTheDocument();
    });

    it("suppresses the team column by default (Free / Primo / single-fleet)", async () => {
      mockedHosts.loadHosts.mockResolvedValue(hosts);

      const { findByText, queryByText } = renderPicker(
        <HostPicker search="col-team-off" onSelect={jest.fn()} />
      );
      // Name + dot still render; team does not.
      expect(await findByText("Rachel's MacBook")).toBeInTheDocument();
      expect(queryByText("Engineering")).not.toBeInTheDocument();
    });
  });
});
