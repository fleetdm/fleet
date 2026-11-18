import { waitFor } from "@testing-library/react";
import { Command } from "cmdk";
import React from "react";

import { IPolicyStats } from "interfaces/policy";
import policiesAPI from "services/entities/policies";
import { createCustomRenderer } from "test/test-utils";

import PolicyPicker from "./PolicyPicker";

// cmdk uses scrollIntoView which JSDOM doesn't implement.
Element.prototype.scrollIntoView = jest.fn();

jest.mock("services/entities/policies", () => ({
  __esModule: true,
  default: { loadAll: jest.fn() },
}));

const mockedPolicies = policiesAPI as jest.Mocked<typeof policiesAPI>;

const renderInClient = createCustomRenderer({ withBackendMock: true });
const renderPicker: typeof renderInClient = (ui, options) =>
  renderInClient(ui, options);
// Some tests render Command.Items, which need a Command root in
// context; the parent dialog supplies one in production.
const renderPickerInCommand = (
  ui: React.ReactElement
): ReturnType<typeof renderInClient> => renderInClient(<Command>{ui}</Command>);

// Minimal IPolicyStats — the picker only reads id, name, type, fleet_id,
// and critical. Cast through `unknown` to skip the full shape.
const policyWith = (
  fields: Partial<IPolicyStats> & Pick<IPolicyStats, "id" | "name">
): IPolicyStats =>
  (({
    fleet_id: null,
    type: "",
    critical: false,
    ...fields,
  } as unknown) as IPolicyStats);

beforeEach(() => {
  mockedPolicies.loadAll.mockReset();
  mockedPolicies.loadAll.mockResolvedValue({ policies: [] });
});

describe("PolicyPicker", () => {
  it("lists All fleets policies without fleet_id when currentTeam is All fleets", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedPolicies.loadAll).toHaveBeenCalledWith(
        expect.objectContaining({ fleetId: undefined, mergeInherited: false })
      );
    });
  });

  it("lists the fleet's policies with mergeInherited when a real team is selected", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedPolicies.loadAll).toHaveBeenCalledWith(
        expect.objectContaining({ fleetId: 5, mergeInherited: true })
      );
    });
  });

  it("lists with fleetId 0 for Unassigned", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 0, name: "No team" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedPolicies.loadAll).toHaveBeenCalledWith(
        expect.objectContaining({ fleetId: 0 })
      );
    });
  });

  it("renders the team-scoped empty state when no team matches", async () => {
    mockedPolicies.loadAll.mockResolvedValue({ policies: [] });

    const { findByText } = renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No policies found in Engineering\./)
    ).toBeInTheDocument();
  });

  it("renders 'in this fleet' empty state when Unassigned is selected", async () => {
    const { findByText } = renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 0, name: "Unassigned" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/No policies found in this fleet\./)
    ).toBeInTheDocument();
  });

  it("renders a fleet-less empty state on All fleets", async () => {
    const { findByText } = renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    expect(await findByText(/^No policies found\.$/)).toBeInTheDocument();
  });

  it("renders a search-specific empty state with query but no fleet suffix on All fleets", async () => {
    const { findByText } = renderPicker(
      <PolicyPicker
        search="missing"
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    expect(
      await findByText(/^No policies match "missing"\.$/)
    ).toBeInTheDocument();
  });

  describe("Patch badge", () => {
    // Unique search values per test sidestep React Query cache pollution
    // from earlier empty-state tests, which registered an empty result
    // under queryKey ["commandPalettePolicies", ..., "<search>"].
    it("renders the Patch badge when policy.type === 'patch'", async () => {
      mockedPolicies.loadAll.mockResolvedValue({
        policies: [
          policyWith({ id: 1, name: "Outdated Chrome", type: "patch" }),
        ],
      });

      const { findByText } = renderPickerInCommand(
        <PolicyPicker
          search="patch-on"
          currentTeam={{ id: -1, name: "All fleets" }}
          onSelect={jest.fn()}
        />
      );

      expect(await findByText("Outdated Chrome")).toBeInTheDocument();
      expect(await findByText("Patch")).toBeInTheDocument();
    });

    it("omits the Patch badge for non-patch policies", async () => {
      mockedPolicies.loadAll.mockResolvedValue({
        policies: [policyWith({ id: 2, name: "Disk encryption", type: "" })],
      });

      const { findByText, queryByText } = renderPickerInCommand(
        <PolicyPicker
          search="patch-off"
          currentTeam={{ id: -1, name: "All fleets" }}
          onSelect={jest.fn()}
        />
      );

      expect(await findByText("Disk encryption")).toBeInTheDocument();
      expect(queryByText("Patch")).not.toBeInTheDocument();
    });
  });
});
