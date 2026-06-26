import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { ActivityType } from "interfaces/activity";
import { IPolicy, IPolicyAutomationActivity } from "interfaces/policy";
import { createCustomRenderer } from "test/test-utils";

import policiesAPI from "services/entities/policies";

import PolicyAutomationsActivitiesTable from "./PolicyAutomationsActivitiesTable";
import { getAutomationRunDisplayName } from "./helpers";

jest.mock("services/entities/policies");

const mockPolicy: IPolicy = {
  id: 123,
  name: "Test policy",
  query: "SELECT 1",
  description: "",
  author_id: 1,
  author_name: "Test",
  author_email: "test@example.com",
  resolution: "",
  platform: "",
  team_id: null,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "custom",
};

const mockActivity = (
  overrides: Partial<IPolicyAutomationActivity> = {}
): IPolicyAutomationActivity => ({
  id: 1,
  created_at: "2026-06-12T15:04:05Z",
  type: ActivityType.InstalledSoftware,
  fleet_initiated: true,
  details: { policy_id: 123, software_title: "1Password" },
  host_id: 42,
  host_display_name: "Anna's MacBook Pro",
  status: "success",
  output: null,
  ...overrides,
});

const mockResponse = (
  activities: IPolicyAutomationActivity[],
  count = activities.length
) => ({
  activities,
  count,
  meta: { has_next_results: false, has_previous_results: false },
});

describe("getAutomationRunDisplayName", () => {
  it("labels software success and failure with the title", () => {
    expect(
      getAutomationRunDisplayName(mockActivity({ status: "success" }))
    ).toBe("Software installed (1Password)");
    expect(getAutomationRunDisplayName(mockActivity({ status: "error" }))).toBe(
      "Software failed (1Password)"
    );
  });

  it("treats App Store (VPP) apps as software", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.InstalledAppStoreApp,
          details: { policy_id: 123, software_title: "Logic Pro" },
        })
      )
    ).toBe("Software installed (Logic Pro)");
  });

  it("labels scripts with the script name", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.RanScript,
          status: "error",
          details: { policy_id: 123, script_name: "remediate.sh" },
        })
      )
    ).toBe("Script failed (remediate.sh)");
  });

  it("labels the named automation types", () => {
    const cases: [
      ActivityType,
      IPolicyAutomationActivity["status"],
      string
    ][] = [
      [
        ActivityType.RanAutomationCalendarEvent,
        "success",
        "Calendar event created",
      ],
      [
        ActivityType.FailedAutomationCalendarEvent,
        "error",
        "Calendar event failed",
      ],
      [
        ActivityType.RanAutomationConditionalAccess,
        "success",
        "Single sign-on blocked",
      ],
      [
        ActivityType.FailedAutomationConditionalAccess,
        "error",
        "Single sign-on failed",
      ],
      [ActivityType.RanAutomationWebhook, "success", "Webhook queued"],
      [ActivityType.FailedAutomationWebhook, "error", "Webhook failed"],
      [ActivityType.RanAutomationTicket, "success", "Ticket queued"],
      [ActivityType.FailedAutomationTicket, "error", "Ticket failed"],
    ];
    cases.forEach(([type, status, label]) => {
      expect(getAutomationRunDisplayName(mockActivity({ type, status }))).toBe(
        label
      );
    });
  });
});

describe("PolicyAutomationsActivitiesTable", () => {
  const render = createCustomRenderer({ withBackendMock: true });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the header, run count, and a host link", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([mockActivity()], 1)
    );

    render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy={false}
      />
    );

    expect(screen.getByText("Automation runs")).toBeInTheDocument();
    expect(await screen.findByText("Anna's MacBook Pro")).toBeInTheDocument();
    expect(screen.getByText("1 run")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Search hosts")).toBeInTheDocument();
  });

  it("shows the Reset policy button only when allowed", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([mockActivity()], 1)
    );

    const { rerender } = render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy={false}
      />
    );
    await screen.findByText("Anna's MacBook Pro");
    expect(
      screen.queryByRole("button", { name: /reset policy/i })
    ).not.toBeInTheDocument();

    rerender(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy
      />
    );
    expect(
      screen.getByRole("button", { name: /reset policy/i })
    ).toBeInTheDocument();
  });

  it("renders the empty state when there are no runs", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([], 0)
    );

    render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy={false}
      />
    );

    expect(await screen.findByText("No automation runs")).toBeInTheDocument();
  });

  it("calls the reset endpoint when the reset is confirmed", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([mockActivity()], 1)
    );
    (policiesAPI.reset as jest.Mock).mockResolvedValue(undefined);

    const { user } = render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy
      />
    );

    await user.click(screen.getByRole("button", { name: "Reset policy" }));
    await user.click(screen.getByRole("button", { name: "Reset" }));

    await waitFor(() => expect(policiesAPI.reset).toHaveBeenCalledWith(123));
  });
});
