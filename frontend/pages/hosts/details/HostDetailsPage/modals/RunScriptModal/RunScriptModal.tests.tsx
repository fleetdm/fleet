import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";
import { createMockHostScript } from "__mocks__/scriptMock";

import RunScriptModal from "./RunScriptModal";

const baseProps = {
  hostTeamId: 7,
  onClose: jest.fn(),
  page: 0,
  setPage: jest.fn(),
  hostScriptResponse: {
    scripts: [],
    meta: { has_next_results: false, has_previous_results: false },
  },
  isFetchingHostScripts: false,
  isLoadingHostScripts: false,
  isError: false,
  onClickViewScript: jest.fn(),
  onClickRunDetails: jest.fn(),
  onClickRun: jest.fn(),
  isRunningScript: false,
  isHidden: false,
};

describe("RunScriptModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("empty state", () => {
    it("shows the 'Add a script' link with fleet_id for a premium admin", () => {
      const adminUser = createMockUser({ global_role: "admin" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: adminUser,
          },
        },
      });

      render(<RunScriptModal {...baseProps} currentUser={adminUser} />);

      expect(screen.getByText("No scripts available")).toBeInTheDocument();
      const link = screen.getByRole("link", { name: /Add a script/i });
      expect(link).toBeInTheDocument();
      expect(link).toHaveAttribute(
        "href",
        expect.stringContaining("fleet_id=7")
      );
      expect(screen.getByText(/available to this host/i)).toBeInTheDocument();
    });

    it("falls back to fleet_id=0 (No team) when the host is unassigned", () => {
      const adminUser = createMockUser({ global_role: "admin" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: adminUser,
          },
        },
      });

      render(
        <RunScriptModal
          {...baseProps}
          currentUser={adminUser}
          hostTeamId={null}
        />
      );

      const link = screen.getByRole("link", { name: /Add a script/i });
      expect(link).toHaveAttribute(
        "href",
        expect.stringContaining("fleet_id=0")
      );
    });

    it("omits fleet_id from the link on free tier", () => {
      const adminUser = createMockUser({ global_role: "admin" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: false,
            currentUser: adminUser,
          },
        },
      });

      render(<RunScriptModal {...baseProps} currentUser={adminUser} />);

      const link = screen.getByRole("link", { name: /Add a script/i });
      expect(link).toBeInTheDocument();
      expect(link.getAttribute("href")).not.toContain("fleet_id");
    });

    it("hides the link and shows guidance text for a global technician", () => {
      const technicianUser = createMockUser({ global_role: "technician" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: technicianUser,
          },
        },
      });

      render(<RunScriptModal {...baseProps} currentUser={technicianUser} />);

      expect(screen.getByText("No scripts available")).toBeInTheDocument();
      expect(
        screen.getByText("Ask your admin to add a script for this host.")
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("link", { name: /Add a script/i })
      ).not.toBeInTheDocument();
    });

    it("shows the link for a team maintainer on the host's team", () => {
      const maintainerUser = createMockUser({
        global_role: null,
        teams: [{ id: 7, name: "Some team", role: "maintainer" }],
      });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: maintainerUser,
          },
        },
      });

      render(<RunScriptModal {...baseProps} currentUser={maintainerUser} />);

      expect(
        screen.getByRole("link", { name: /Add a script/i })
      ).toBeInTheDocument();
    });

    it("hides the link for a team technician on the host's team", () => {
      const technicianUser = createMockUser({
        global_role: null,
        teams: [{ id: 7, name: "Some team", role: "technician" }],
      });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: technicianUser,
          },
        },
      });

      render(<RunScriptModal {...baseProps} currentUser={technicianUser} />);

      expect(
        screen.getByText("Ask your admin to add a script for this host.")
      ).toBeInTheDocument();
      expect(
        screen.queryByRole("link", { name: /Add a script/i })
      ).not.toBeInTheDocument();
    });
  });

  describe("table state", () => {
    it("renders the script table instead of the empty state when scripts exist", () => {
      const adminUser = createMockUser({ global_role: "admin" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: adminUser,
          },
        },
      });

      render(
        <RunScriptModal
          {...baseProps}
          currentUser={adminUser}
          hostScriptResponse={{
            scripts: [createMockHostScript({ name: "cleanup.sh" })],
            meta: { has_next_results: false, has_previous_results: false },
          }}
        />
      );

      expect(
        screen.queryByText("No scripts available")
      ).not.toBeInTheDocument();
      // TooltipTruncatedTextCell renders the script name in both the cell and tooltip
      expect(screen.getAllByText("cleanup.sh").length).toBeGreaterThan(0);
    });
  });

  describe("error and loading states", () => {
    it("renders DataError when isError is true", () => {
      const adminUser = createMockUser({ global_role: "admin" });
      const render = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            config: createMockConfig(),
            isPremiumTier: true,
            currentUser: adminUser,
          },
        },
      });

      render(
        <RunScriptModal
          {...baseProps}
          currentUser={adminUser}
          isError
          hostScriptResponse={undefined}
        />
      );

      expect(
        screen.queryByText("No scripts available")
      ).not.toBeInTheDocument();
      expect(screen.getByText(/something's gone wrong/i)).toBeInTheDocument();
    });
  });
});
