import React from "react";

import {
  createMockSoftwareTitle,
  createMockSoftwareTitleDetails,
  createMockAppStoreApp,
} from "__mocks__/softwareMock";

import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { userEvent } from "@testing-library/user-event";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { ILabelSummary } from "interfaces/label";

import createMockUser from "__mocks__/userMock";
import createMockConfig from "__mocks__/configMock";

import EditAutoUpdateConfigModal from "./EditAutoUpdateConfigModal";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const mockLabels: ILabelSummary[] = [
  {
    id: 1,
    name: "Fun",
    description: "Computers that like to have a good time",
    label_type: "regular",
  },
  {
    id: 2,
    name: "Fresh",
    description: "Laptops with dirty mouths",
    label_type: "regular",
  },
];

const labelSummariesHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({
    labels: mockLabels,
  });
});

describe("Edit Auto Update Config Modal", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        currentUser: createMockUser(),
        isGlobalObserver: false,
        isGlobalAdmin: true,
        isGlobalMaintainer: false,
        isOnGlobalTeam: true,
        isPremiumTier: true,
        isSandboxMode: false,
        // config: createMockConfig(),
      },
    },
  });
  describe("Auto updates options", () => {
    it("Does not show maintenance window options when 'Enable auto updates' is not configured", async () => {
      render(
        <EditAutoUpdateConfigModal
          softwareTitle={createMockSoftwareTitleDetails()}
          teamId={1}
          refetchSoftwareTitle={jest.fn()}
          onExit={jest.fn()}
        />
      );
      // Verify that "Enable auto updates" checkbox is not checked.
      const enableAutoUpdatesCheckbox = screen.getByRole("checkbox", {
        name: "Enable auto updates",
      });
      expect(enableAutoUpdatesCheckbox).not.toBeChecked();
      // Verify that the maintenance window fields are not shown.
      expect(
        screen.queryByLabelText("Earliest start time")
      ).not.toBeInTheDocument();
      expect(
        screen.queryByLabelText("Latest start time")
      ).not.toBeInTheDocument();
    });

    it("Shows maintenance window options when 'Enable auto updates' is configured", async () => {
      render(
        <EditAutoUpdateConfigModal
          softwareTitle={createMockSoftwareTitleDetails({
            auto_update_enabled: true,
            auto_update_start_time: "02:00",
            auto_update_end_time: "04:00",
          })}
          teamId={1}
          refetchSoftwareTitle={jest.fn()}
          onExit={jest.fn()}
        />
      );
      // Verify that "Enable auto updates" checkbox is not checked.
      const enableAutoUpdatesCheckbox = screen.getByRole("checkbox", {
        name: "Enable auto updates",
      });
      expect(enableAutoUpdatesCheckbox).toBeChecked();
      // Verify that the maintenance window fields are shown correctly.
      const startTimeField = screen.getByLabelText("Earliest start time");
      const endTimeField = screen.getByLabelText("Latest start time");
      expect(startTimeField).toBeInTheDocument();
      expect(startTimeField).toHaveValue("02:00");
      expect(endTimeField).toBeInTheDocument();
      expect(endTimeField).toHaveValue("04:00");
    });

    it("Shows maintenance window options when 'Enable auto updates' is checked", async () => {
      const { user } = render(
        <EditAutoUpdateConfigModal
          softwareTitle={createMockSoftwareTitleDetails()}
          teamId={1}
          refetchSoftwareTitle={jest.fn()}
          onExit={jest.fn()}
        />
      );
      // Verify that "Enable auto updates" checkbox is not checked.
      const enableAutoUpdatesCheckbox = screen.getByRole("checkbox", {
        name: "Enable auto updates",
      });
      expect(enableAutoUpdatesCheckbox).not.toBeChecked();
      // Click the checkbox to enable auto updates.
      await user.click(enableAutoUpdatesCheckbox);
      await waitFor(() => {
        expect(enableAutoUpdatesCheckbox).toBeChecked();
        // Verify that the maintenance window fields are shown.
        const startTimeField = screen.getByLabelText("Earliest start time");
        const endTimeField = screen.getByLabelText("Latest start time");
        expect(startTimeField).toBeInTheDocument();
        expect(endTimeField).toBeInTheDocument();
      });
    });

    it("Hides maintenance window options when 'Enable auto updates' is unchecked", async () => {
      const { user } = render(
        <EditAutoUpdateConfigModal
          softwareTitle={createMockSoftwareTitleDetails({
            auto_update_enabled: true,
            auto_update_start_time: "02:00",
            auto_update_end_time: "04:00",
          })}
          teamId={1}
          refetchSoftwareTitle={jest.fn()}
          onExit={jest.fn()}
        />
      );
      // Verify that "Enable auto updates" checkbox is not checked.
      const enableAutoUpdatesCheckbox = screen.getByRole("checkbox", {
        name: "Enable auto updates",
      });
      expect(enableAutoUpdatesCheckbox).toBeChecked();
      // Click the checkbox to enable auto updates.
      await user.click(enableAutoUpdatesCheckbox);
      await waitFor(() => {
        expect(enableAutoUpdatesCheckbox).not.toBeChecked();
        // Verify that the maintenance window fields are shown.
        const startTimeField = screen.queryByText("Earliest start time");
        const endTimeField = screen.queryByText("Latest start time");
        expect(startTimeField).not.toBeInTheDocument();
        expect(endTimeField).not.toBeInTheDocument();
      });
    });

    describe("Maintenance window validation", () => {
      it("Requires start time to be HH:MM format", async () => {});
      it("Requires end time to be HH:MM format", async () => {});
      it("Requires both start and end times to be set", async () => {});
      it("Requires window to be at least one hour", async () => {});
    });
  });
  describe("Target options", () => {
    it("Shows 'All hosts' if no labels are configured for the title", async () => {});
    it("Shows label options if labels are configured for the title", async () => {});
  });
  describe("Submitting the form", () => {
    it("Sends the correct payload when 'Enable auto updates' is unchecked", async () => {});
    it("Sends the correct payload when 'Enable auto updates' is checked and a valid window is configured", async () => {});
    it("Sends the correct payload when 'All hosts' is selected as the target", async () => {});
    it("Sends the correct payload when specific labels are selected as the target", async () => {});
  });
});
