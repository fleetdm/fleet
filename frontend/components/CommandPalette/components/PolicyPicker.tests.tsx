import React from "react";
import { waitFor } from "@testing-library/react";
import { Command } from "cmdk";
import { createCustomRenderer } from "test/test-utils";

import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";
import { IPolicyStats } from "interfaces/policy";

import PolicyPicker from "./PolicyPicker";

// cmdk uses scrollIntoView which JSDOM doesn't implement.
Element.prototype.scrollIntoView = jest.fn();

jest.mock("services/entities/global_policies", () => ({
  __esModule: true,
  default: { loadAllNew: jest.fn() },
}));
jest.mock("services/entities/team_policies", () => ({
  __esModule: true,
  default: { loadAllNew: jest.fn() },
}));

const mockedGlobal = globalPoliciesAPI as jest.Mocked<typeof globalPoliciesAPI>;
const mockedTeam = teamPoliciesAPI as jest.Mocked<typeof teamPoliciesAPI>;

const renderInClient = createCustomRenderer({ withBackendMock: true });
const renderPicker: typeof renderInClient = (ui, options) =>
  renderInClient(ui, options);
// Some tests render Command.Items, which need a Command root in
// context; the parent dialog supplies one in production.
const renderPickerInCommand = (
  ui: React.ReactElement
): ReturnType<typeof renderInClient> => renderInClient(<Command>{ui}</Command>);

// Minimal IPolicyStats — the picker only reads id, name, type, team_id,
// and critical. Cast through `unknown` to skip the full shape.
const policyWith = (
  fields: Partial<IPolicyStats> & Pick<IPolicyStats, "id" | "name">
): IPolicyStats =>
  (({
    team_id: null,
    type: "",
    critical: false,
    ...fields,
  } as unknown) as IPolicyStats);

beforeEach(() => {
  mockedGlobal.loadAllNew.mockReset();
  mockedTeam.loadAllNew.mockReset();
  mockedGlobal.loadAllNew.mockResolvedValue({ policies: [] });
  mockedTeam.loadAllNew.mockResolvedValue({ policies: [] });
});

describe("PolicyPicker", () => {
  it("calls globalPoliciesAPI when currentTeam is All fleets", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: -1, name: "All fleets" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedGlobal.loadAllNew).toHaveBeenCalled();
    });
    expect(mockedTeam.loadAllNew).not.toHaveBeenCalled();
  });

  it("calls teamPoliciesAPI with mergeInherited when a real team is selected", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 5, name: "Engineering" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedTeam.loadAllNew).toHaveBeenCalledWith(
        expect.objectContaining({ teamId: 5, mergeInherited: true })
      );
    });
    expect(mockedGlobal.loadAllNew).not.toHaveBeenCalled();
  });

  it("calls teamPoliciesAPI with teamId 0 for Unassigned", async () => {
    renderPicker(
      <PolicyPicker
        search=""
        currentTeam={{ id: 0, name: "No team" }}
        onSelect={jest.fn()}
      />
    );

    await waitFor(() => {
      expect(mockedTeam.loadAllNew).toHaveBeenCalledWith(
        expect.objectContaining({ teamId: 0 })
      );
    });
  });

  it("renders the team-scoped empty state when no team matches", async () => {
    mockedTeam.loadAllNew.mockResolvedValue({ policies: [] });

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
      mockedGlobal.loadAllNew.mockResolvedValue({
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
      mockedGlobal.loadAllNew.mockResolvedValue({
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
