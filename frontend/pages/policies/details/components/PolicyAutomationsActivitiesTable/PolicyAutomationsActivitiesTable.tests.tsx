import { screen, waitFor } from "@testing-library/react";
import React from "react";

import {
  PRE_INSTALL_QUERY_FAIL_OUTPUT,
  SKIPPED_INSTALL_DETAILS,
} from "components/ActivityDetails/InstallDetails/constants";
import { SKIPPED_INSTALL_NOTIFY_EXPLANATION } from "components/ActivityDetails/NotifyBeforePatchingDetailsModal/helpers";
import { ActivityType } from "interfaces/activity";
import { IPolicy, IPolicyAutomationActivity } from "interfaces/policy";
import policiesAPI from "services/entities/policies";
import { createCustomRenderer } from "test/test-utils";

import {
  getAutomationRunDisplayName,
  getAutomationStatusIcon,
  getDetailOutputText,
} from "./helpers";
import PolicyAutomationsActivitiesTable from "./PolicyAutomationsActivitiesTable";

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
  pre_install_output: null,
  post_install_output: null,
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

  it("labels a patch-when-closed skip as skipped, not failed", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          status: "error",
          details: {
            policy_id: 123,
            software_title: "1Password",
            skipped_install: true,
          },
        })
      )
    ).toBe("Patch skipped (1Password)");
  });

  it("labels a successful notify as 'Notified end user' and a failure as 'Failed to notify'", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
          details: { policy_id: 123, software_title: "1Password" },
        })
      )
    ).toBe("Notified end user (1Password)");
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "error",
          details: { policy_id: 123, software_title: "1Password" },
        })
      )
    ).toBe("Failed to notify (1Password)");
  });

  it("falls back to software_titles[0] on notify rows when software_title is absent and no policy scope is passed", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
          details: {
            policy_id: 123,
            software_titles: ["1Password", "Slack"],
          },
        })
      )
    ).toBe("Notified end user (1Password)");
  });

  it("scopes a multi-title notify to the title paired with the current policy", () => {
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
          details: {
            software_titles: ["1Password", "Slack"],
            policy_ids: [222, 123],
          },
        }),
        123
      )
    ).toBe("Notified end user (Slack)");
  });

  it("falls back to the first title when policy_ids and software_titles are desynced", () => {
    // A deleted policy leaves patch_notification_apps.policy_id NULL, so the
    // BE emits fewer policy_ids than software_titles and index-alignment is
    // no longer safe. Fall back to the first title rather than mismatching.
    expect(
      getAutomationRunDisplayName(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
          details: {
            software_titles: ["1Password", "Slack"],
            policy_ids: [123],
          },
        }),
        123
      )
    ).toBe("Notified end user (1Password)");
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

describe("getAutomationStatusIcon", () => {
  it("uses a muted grey error glyph for a skip, red for other failures, green for success", () => {
    expect(
      getAutomationStatusIcon(
        mockActivity({
          status: "error",
          details: {
            policy_id: 123,
            software_title: "1Password",
            skipped_install: true,
          },
        })
      )
    ).toEqual({ name: "error-outline", color: "ui-fleet-black-50" });
    expect(getAutomationStatusIcon(mockActivity({ status: "error" }))).toEqual({
      name: "error-outline",
    });
    expect(
      getAutomationStatusIcon(mockActivity({ status: "success" }))
    ).toEqual({ name: "success-outline" });
  });

  it("uses the grey error glyph for a successful 'end user notified' row and red for the failure row", () => {
    expect(
      getAutomationStatusIcon(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
        })
      )
    ).toEqual({ name: "error-outline", color: "ui-fleet-black-50" });
    expect(
      getAutomationStatusIcon(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "error",
        })
      )
    ).toEqual({ name: "error-outline" });
  });
});

describe("getDetailOutputText for notify rows", () => {
  it("renders the 1-hour sentence when time_before is 3600", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "success",
          details: {
            policy_id: 123,
            software_title: "1Password",
            time_before: 3600,
          },
        })
      )
    ).toMatch(/Patch will be forced in 1 hour\./);
  });

  it("renders the 5-minute sentence when time_before is 300 (reminder)", () => {
    const text = getDetailOutputText(
      mockActivity({
        type: ActivityType.NotifiedEndUserBeforePatching,
        status: "success",
        details: {
          policy_id: 123,
          software_title: "1Password",
          time_before: 300,
        },
      })
    );
    expect(text).toMatch(/Patch will be forced in 5 minutes\./);
    // The recovery-cadence tail is a system default and stays "1 hour".
    expect(text).toMatch(/patches it after 1 hour/);
  });

  it("falls through to activity.output for a notify failure row", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          type: ActivityType.NotifiedEndUserBeforePatching,
          status: "error",
          output: "screen was locked",
          details: {
            policy_id: 123,
            software_title: "1Password",
            time_before: 3600,
          },
        })
      )
    ).toBe("screen was locked");
  });
});

describe("getDetailOutputText", () => {
  it("explains a patch-when-closed skip rather than returning empty text", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "error",
          pre_install_output: "",
          details: {
            policy_id: 123,
            software_title: "1Password",
            skipped_install: true,
            patch_when_closed: true,
          },
        })
      )
    ).toBe(SKIPPED_INSTALL_DETAILS);
  });

  it("explains a notify-before-patching skip with the notification sentence", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "error",
          pre_install_output: "",
          details: {
            policy_id: 123,
            software_title: "1Password",
            skipped_install: true,
            patch_when_closed: false,
          },
        })
      )
    ).toBe(SKIPPED_INSTALL_NOTIFY_EXPLANATION);
  });

  it("treats a skip recorded before patch_when_closed existed as patch-when-closed", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "error",
          pre_install_output: "",
          details: {
            policy_id: 123,
            software_title: "1Password",
            skipped_install: true,
          },
        })
      )
    ).toBe(SKIPPED_INSTALL_DETAILS);
  });

  it("reports the query-fail copy for an install stopped by its pre-install query", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "error",
          pre_install_output: "",
          details: { policy_id: 123, software_title: "1Password" },
        })
      )
    ).toBe(PRE_INSTALL_QUERY_FAIL_OUTPUT);
  });

  it("returns empty text when no pre-install query was configured", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "error",
          pre_install_output: null,
          details: { policy_id: 123, software_title: "1Password" },
        })
      )
    ).toBe("");
  });

  it("returns empty text for a successful install even though its pre-install output is empty", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          status: "success",
          pre_install_output: "",
          details: { policy_id: 123, software_title: "1Password" },
        })
      )
    ).toBe("");
  });

  it("does not report the query-fail copy for a non-install activity even with an empty pre-install output", () => {
    expect(
      getDetailOutputText(
        mockActivity({
          type: ActivityType.RanScript,
          status: "error",
          pre_install_output: "",
          details: { policy_id: 123, script_name: "remediate.sh" },
        })
      )
    ).toBe("");
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

  it("renders one row per host for a batch activity sharing an activity id", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([
        mockActivity({
          id: 41,
          type: ActivityType.RanAutomationWebhook,
          details: { policy_id: 123 },
          host_id: 1,
          host_display_name: "batch-host-a",
        }),
        mockActivity({
          id: 41,
          type: ActivityType.RanAutomationWebhook,
          details: { policy_id: 123 },
          host_id: 2,
          host_display_name: "batch-host-b",
        }),
      ])
    );

    render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy={false}
      />
    );

    // Both (activity, host) rows must render even though they share id 41.
    expect(await screen.findByText("batch-host-a")).toBeInTheDocument();
    expect(screen.getByText("batch-host-b")).toBeInTheDocument();
    expect(screen.getByText("2 runs")).toBeInTheDocument();
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

  it("resets the whole policy from the header button", async () => {
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

    await waitFor(() =>
      expect(policiesAPI.reset).toHaveBeenCalledWith(123, undefined)
    );
  });

  it("resets the policy only for the run's host when opened from a run", async () => {
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

    await user.click(await screen.findByText("Software installed (1Password)"));
    // The header also has a "Reset policy" button; the run's modal renders last.
    const resetButtons = screen.getAllByRole("button", {
      name: "Reset policy",
    });
    await user.click(resetButtons[resetButtons.length - 1]);
    expect(
      screen.getByText("Anna's MacBook Pro", { selector: "b" })
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reset" }));

    await waitFor(() =>
      expect(policiesAPI.reset).toHaveBeenCalledWith(123, 42)
    );
  });

  it("keeps a host-scoped reset when the run's host has no display name", async () => {
    (policiesAPI.getAutomationActivities as jest.Mock).mockResolvedValue(
      mockResponse([mockActivity({ host_display_name: "" })], 1)
    );
    (policiesAPI.reset as jest.Mock).mockResolvedValue(undefined);

    const { user } = render(
      <PolicyAutomationsActivitiesTable
        policy={mockPolicy}
        currentAutomatedPolicies={[]}
        canResetPolicy
      />
    );

    await user.click(await screen.findByText("Software installed (1Password)"));
    const resetButtons = screen.getAllByRole("button", {
      name: "Reset policy",
    });
    await user.click(resetButtons[resetButtons.length - 1]);
    expect(
      screen.getByText(/this host until its next check in/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/all hosts/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Reset" }));

    await waitFor(() =>
      expect(policiesAPI.reset).toHaveBeenCalledWith(123, 42)
    );
  });
});
