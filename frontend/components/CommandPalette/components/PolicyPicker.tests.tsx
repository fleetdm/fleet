import React from "react";
import { waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import globalPoliciesAPI from "services/entities/global_policies";
import teamPoliciesAPI from "services/entities/team_policies";

import PolicyPicker from "./PolicyPicker";

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

const renderPicker = createCustomRenderer({ withBackendMock: true });

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
});
