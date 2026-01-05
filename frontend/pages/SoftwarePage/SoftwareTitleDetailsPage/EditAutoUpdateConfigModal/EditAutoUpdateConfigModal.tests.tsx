import React from "react";

import {
  createMockSoftwareTitle,
  createMockSoftwareTitleDetails,
  createMockAppStoreApp,
} from "__mocks__/softwareMock";

import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
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
  beforeEach(() => {
    mockServer.use(labelSummariesHandler);
  });

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
        expect(startTimeField).toHaveValue("");
        expect(endTimeField).toHaveValue("");
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
      it("Requires start time to be HH:MM format", async () => {
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
        });
        const startTimeField = screen.getByLabelText("Earliest start time");
        let endTimeField = screen.getByLabelText("Latest start time");
        expect(startTimeField).toBeInTheDocument();
        expect(endTimeField).toBeInTheDocument();
        // Enter invalid start time
        await user.type(startTimeField, "19:99");
        // Move focus to trigger validation
        await user.click(endTimeField);
        await user.type(endTimeField, "12:00");
        // Verify that validation message is shown
        const errorField = screen.getByLabelText(
          "Use HH:MM format (24-hour clock)"
        );
        expect(errorField).toBeInTheDocument();
        expect(errorField).toHaveValue("19:99");
        // Veryfy that end time is still present with valid label.
        endTimeField = screen.getByLabelText("Latest start time");
        expect(endTimeField).toBeInTheDocument();
        expect(endTimeField).toHaveValue("12:00");
      });

      it("Requires end time to be HH:MM format", async () => {
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
        });
        let startTimeField = screen.getByLabelText("Earliest start time");
        const endTimeField = screen.getByLabelText("Latest start time");
        expect(startTimeField).toBeInTheDocument();
        expect(endTimeField).toBeInTheDocument();
        // Enter invalid start time
        await user.type(endTimeField, "19:99");
        // Move focus to trigger validation
        await user.click(startTimeField);
        await user.type(startTimeField, "12:00");
        // Verify that validation message is shown
        const errorField = screen.getByLabelText(
          "Use HH:MM format (24-hour clock)"
        );
        expect(errorField).toBeInTheDocument();
        expect(errorField).toHaveValue("19:99");
        // Veryfy that end time is still present with valid label.
        startTimeField = screen.getByLabelText("Earliest start time");
        expect(startTimeField).toBeInTheDocument();
        expect(startTimeField).toHaveValue("12:00");
      });

      it("Requires both start and end times to be set", async () => {
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
        });
        const startTimeField = screen.getByLabelText("Earliest start time");
        const endTimeField = screen.getByLabelText("Latest start time");
        const saveButton = screen.getByRole("button", { name: "Save" });

        expect(startTimeField).toBeInTheDocument();
        expect(endTimeField).toBeInTheDocument();
        // Enter only start time
        await user.type(startTimeField, "10:00");
        // Click Save button to trigger validation
        await user.click(saveButton);
        // Verify that validation message is shown for end time
        expect(
          screen.getByLabelText("Latest start time is required")
        ).toBeInTheDocument();
        // Now enter only end time
        await user.clear(startTimeField);
        await user.type(endTimeField, "12:00");
        // Click Save button to trigger validation
        await user.click(saveButton);
        expect(
          screen.getByLabelText("Earliest start time is required")
        ).toBeInTheDocument();
        expect(
          screen.queryByText("Latest start time is required")
        ).not.toBeInTheDocument();
        // Clear both
        await user.clear(startTimeField);
        await user.clear(endTimeField);
        // Click Save button to trigger validation
        await user.click(saveButton);
        // Verify that validation message is shown for start time
        expect(
          screen.getByLabelText("Earliest start time is required")
        ).toBeInTheDocument();
        expect(
          screen.getByLabelText("Latest start time is required")
        ).toBeInTheDocument();
      });

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
