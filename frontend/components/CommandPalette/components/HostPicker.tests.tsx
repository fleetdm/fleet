import React from "react";
import { waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import hostsAPI, { ILoadHostsResponse } from "services/entities/hosts";

import HostPicker from "./HostPicker";

jest.mock("services/entities/hosts", () => ({
  __esModule: true,
  default: { loadHosts: jest.fn() },
}));

const mockedHosts = hostsAPI as jest.Mocked<typeof hostsAPI>;

const renderPicker = createCustomRenderer({ withBackendMock: true });

// Minimal valid response — the picker only reads `hosts`, so we cast
// through `unknown` rather than constructing the unused munki/MDM
// aggregates. The typed local lets future renames of the field this
// test cares about surface here.
const emptyHostsResponse: ILoadHostsResponse = ({
  hosts: [],
} as unknown) as ILoadHostsResponse;

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
});
