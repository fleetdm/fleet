import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import { createMockVppToken } from "__mocks__/mdmMock";
import { createMockTeamSummary } from "__mocks__/teamMock";
import { IMdmVppToken } from "interfaces/mdm";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import EditTeamsVppModal from "./EditTeamsVppModal";

const makeToken = (teams: IMdmVppToken["teams"]): IMdmVppToken =>
  createMockVppToken({ id: 7, teams });

const availableTeams = [
  createMockTeamSummary({ id: -1, name: "All fleets" }),
  createMockTeamSummary({ id: 1, name: "Workstations" }),
  createMockTeamSummary({ id: 2, name: "Personal mobile devices" }),
  createMockTeamSummary({ id: 0, name: "Unassigned" }),
];

const render = createCustomRenderer({
  withBackendMock: true,
  context: { app: { availableTeams } },
});

const mockPatch = () => {
  const bodies: unknown[] = [];
  mockServer.use(
    http.patch(baseUrl("/vpp_tokens/7/fleets"), async ({ request }) => {
      bodies.push(await request.json());
      return HttpResponse.json({});
    })
  );
  return bodies;
};

const renderModal = (token: IMdmVppToken, onSuccess = jest.fn()) =>
  render(
    <EditTeamsVppModal
      currentToken={token}
      onCancel={jest.fn()}
      onSuccess={onSuccess}
    />
  );

describe("EditTeamsVppModal", () => {
  it("renders the intro and none of the one-token-per-fleet copy", () => {
    renderModal(makeToken([]));

    expect(screen.getByText("Acme Inc.")).toBeInTheDocument();
    expect(
      screen.queryByText(/Each fleet can have only one VPP token/)
    ).toBeNull();
    expect(screen.queryByText(/If you delete a fleet/)).toBeNull();
    expect(screen.queryByText(/You can't choose fleets because/)).toBeNull();
  });

  describe("initial radio state", () => {
    it("selects All fleets for a token on all fleets", () => {
      renderModal(makeToken([]));

      expect(screen.getByRole("radio", { name: "All fleets" })).toBeChecked();
      expect(screen.queryByRole("checkbox")).toBeNull();
    });

    it("selects Custom with the token's fleets checked", () => {
      renderModal(
        makeToken([
          { team_id: 2, name: "Personal mobile devices" },
          { team_id: 0, name: "No team" },
        ])
      );

      expect(screen.getByRole("radio", { name: "Custom" })).toBeChecked();
      expect(
        screen.getByRole("checkbox", { name: "Personal mobile devices" })
      ).toBeChecked();
      expect(
        screen.getByRole("checkbox", { name: "Unassigned" })
      ).toBeChecked();
      expect(
        screen.getByRole("checkbox", { name: "Workstations" })
      ).not.toBeChecked();
    });

    it("selects Custom with nothing checked for an unassigned token", () => {
      renderModal(makeToken(null));

      expect(screen.getByRole("radio", { name: "Custom" })).toBeChecked();
      screen
        .getAllByRole("checkbox")
        .forEach((box) => expect(box).not.toBeChecked());
    });
  });

  it("lists a fleet that another token already holds", () => {
    // The modal never receives other tokens, so every fleet is listed.
    renderModal(makeToken(null));

    expect(
      screen.getByRole("checkbox", { name: "Workstations" })
    ).toBeInTheDocument();
  });

  describe("save", () => {
    it("sends [] for All fleets", async () => {
      const bodies = mockPatch();
      const onSuccess = jest.fn();
      const { user } = renderModal(makeToken(null), onSuccess);

      await user.click(screen.getByRole("radio", { name: "All fleets" }));
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => expect(onSuccess).toHaveBeenCalled());
      expect(bodies).toEqual([{ fleets: [] }]);
    });

    it("sends the checked fleet ids for Custom", async () => {
      const bodies = mockPatch();
      const onSuccess = jest.fn();
      const { user } = renderModal(makeToken(null), onSuccess);

      await user.click(screen.getByRole("checkbox", { name: "Workstations" }));
      await user.click(
        screen.getByRole("checkbox", { name: "Personal mobile devices" })
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => expect(onSuccess).toHaveBeenCalled());
      expect(bodies).toEqual([{ fleets: [1, 2] }]);
    });

    it("sends null for Custom with nothing checked", async () => {
      const bodies = mockPatch();
      const onSuccess = jest.fn();
      const { user } = renderModal(
        makeToken([{ team_id: 1, name: "W" }]),
        onSuccess
      );

      await user.click(
        screen.getByRole("button", { name: "Remove Workstations" })
      );
      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() => expect(onSuccess).toHaveBeenCalled());
      expect(bodies).toEqual([{ fleets: null }]);
    });
  });
});
