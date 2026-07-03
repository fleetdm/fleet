import React from "react";
import {
  getInstallSoftwareErrorMessage,
  getRunScriptErrorMessage,
  getAutomationsForPolicy,
  getPageAfterDelete,
} from "./helpers";
import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";
import { IPolicyRunScriptFormData } from "./components/PolicyRunScriptModal/PolicyRunScriptModal";

describe("getInstallSoftwareErrorMessage", () => {
  const mockFormData: IInstallSoftwareFormData = [
    {
      swIdToInstall: 178,
      swNameToInstall: "Keynote.app",
      name: "Test policy",
      id: 1,
      installSoftwareEnabled: true,
      platform: "darwin",
      runScriptEnabled: false,
      query: "",
      description: "",
      author_id: 0,
      author_name: "",
      author_email: "",
      resolution: "",
      team_id: null,
      created_at: "",
      updated_at: "",
      critical: false,
      calendar_events_enabled: false,
      conditional_access_enabled: false,
      type: "dynamic",
    },
    {
      swIdToInstall: 456,
      swNameToInstall: "Another Software",
      name: "Another test policy",
      id: 2,
      installSoftwareEnabled: true,
      platform: "darwin",
      runScriptEnabled: false,
      query: "",
      description: "",
      author_id: 0,
      author_name: "",
      author_email: "",
      resolution: "",
      team_id: null,
      created_at: "",
      updated_at: "",
      critical: false,
      calendar_events_enabled: false,
      conditional_access_enabled: false,
      type: "dynamic",
    },
  ];

  const renderToString = (element: JSX.Element): string => {
    return React.Children.toArray(element.props.children)
      .map((child) => {
        if (typeof child === "string") return child;
        if (React.isValidElement(child)) {
          return renderToString(child);
        }
        return "";
      })
      .join("");
  };

  it("returns a JSX element with the correct error message for software and team ", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason:
                "Software title with ID 178 on team ID 789 does not have associated package",
            },
          ],
        },
      },
      status: "rejected",
    };

    const currentTeamName = "1a - Workstations (canary)";

    const result = getInstallSoftwareErrorMessage(
      mockResult,
      mockFormData,
      currentTeamName
    );
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Keynote.app (ID: 178) on 1a - Workstations (canary) does not have associated package"
    );
  });

  it("handles unknown software title id", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with software title with ID 999",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getInstallSoftwareErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with software title with ID 999"
    );
  });

  it("handles missing currentTeamName", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with team ID 789",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getInstallSoftwareErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with team ID 789"
    );
  });
});

describe("getRunScriptErrorMessage", () => {
  const mockFormData: IPolicyRunScriptFormData = [
    {
      scriptIdToRun: 123,
      scriptNameToRun: "Test Script",
      name: "Test policy",
      id: 1,
      installSoftwareEnabled: false,
      platform: "darwin",
      runScriptEnabled: true,
      query: "",
      description: "",
      author_id: 0,
      author_name: "",
      author_email: "",
      resolution: "",
      team_id: null,
      created_at: "",
      updated_at: "",
      critical: false,
      calendar_events_enabled: false,
      conditional_access_enabled: false,
      type: "dynamic",
    },
    {
      scriptIdToRun: 456,
      scriptNameToRun: "Another Script",
      name: "Another test policy",
      id: 2,
      installSoftwareEnabled: false,
      platform: "darwin",
      runScriptEnabled: true,
      query: "",
      description: "",
      author_id: 0,
      author_name: "",
      author_email: "",
      resolution: "",
      team_id: null,
      created_at: "",
      updated_at: "",
      critical: false,
      calendar_events_enabled: false,
      conditional_access_enabled: false,
      type: "dynamic",
    },
  ];

  const renderToString = (element: JSX.Element): string => {
    return React.Children.toArray(element.props.children)
      .map((child) => {
        if (typeof child === "string") return child;
        if (React.isValidElement(child)) {
          return renderToString(child);
        }
        return "";
      })
      .join("");
  };

  it("returns a JSX element with the correct error message for script and team", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Script with ID 123 does not belong to team ID 789",
            },
          ],
        },
      },
      status: "rejected",
    };

    const currentTeamName = "1a - Workstations (canary)";

    const result = getRunScriptErrorMessage(
      mockResult,
      mockFormData,
      currentTeamName
    );
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Test Script (ID: 123) does not belong to 1a - Workstations (canary)"
    );
  });

  it("handles unknown script id", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with script with ID 999",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getRunScriptErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with script with ID 999"
    );
  });

  it("handles missing currentTeamName", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with team ID 789",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getRunScriptErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with team ID 789"
    );
  });
});

describe("getAutomationsForPolicy", () => {
  const basePolicy = {
    calendar_events_enabled: false,
    conditional_access_enabled: false,
    webhook: "Off",
  };

  it("returns empty array when no automations are enabled", () => {
    expect(getAutomationsForPolicy(basePolicy)).toEqual([]);
  });

  it("returns software automation with display_name preferred over name", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: {
        name: "Chrome.app",
        display_name: "Google Chrome",
        software_title_id: 42,
      },
    });
    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      type: "software",
      name: "Google Chrome",
      softwareTitleId: 42,
    });
  });

  it("carries the custom icon_url onto the software automation", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: {
        name: "Chrome.app",
        software_title_id: 42,
        icon_url: "/api/latest/fleet/software/titles/42/icon?fleet_id=1",
      },
    });
    expect(result[0]).toMatchObject({
      type: "software",
      iconUrl: "/api/latest/fleet/software/titles/42/icon?fleet_id=1",
    });
  });

  it("leaves iconUrl undefined when no custom icon exists", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: { name: "Chrome.app", software_title_id: 42 },
    });
    expect(result[0]).toMatchObject({ type: "software" });
    expect((result[0] as { iconUrl?: string | null }).iconUrl).toBeUndefined();
  });

  it("falls back to name when display_name is absent", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: { name: "Chrome.app", software_title_id: 42 },
    });
    expect(result[0].name).toBe("Chrome.app");
  });

  it("normalizes known awkward titles via getDisplayedSoftwareName", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: {
        name: "Microsoft.CompanyPortal",
        software_title_id: 42,
      },
    });
    expect(result[0].name).toBe("Company Portal");
  });

  it("preserves the raw install_software.name as iconName for fallback icon matching (regression: #47123)", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      install_software: {
        name: "Zoom",
        display_name: "Custom Renamed App",
        software_title_id: 42,
      },
    });
    expect(result[0]).toMatchObject({
      type: "software",
      name: "Custom Renamed App",
      iconName: "Zoom",
    });
  });

  it("returns script automation with file name", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      run_script: { id: 7, name: "fix-disk.sh" },
    });
    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      type: "script",
      name: "fix-disk.sh",
    });
  });

  it("returns calendar automation", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      calendar_events_enabled: true,
    });
    expect(result[0]).toMatchObject({
      type: "calendar",
      name: "Maintenance window",
    });
  });

  it("returns conditional access automation", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      conditional_access_enabled: true,
    });
    expect(result[0]).toMatchObject({
      type: "conditional_access",
      name: "Conditional access",
    });
  });

  it("labels other automation as Webhook by default", () => {
    const result = getAutomationsForPolicy({ ...basePolicy, webhook: "On" });
    expect(result[0]).toMatchObject({ type: "other", name: "Webhook" });
  });

  it("labels other automation as Ticket when otherAutomationType is ticket", () => {
    const result = getAutomationsForPolicy(
      { ...basePolicy, webhook: "On" },
      "ticket"
    );
    expect(result[0]).toMatchObject({ type: "other", name: "Ticket" });
  });

  it("does not include other automation when webhook is Off", () => {
    const result = getAutomationsForPolicy({
      ...basePolicy,
      webhook: "Off",
    });
    expect(result).toHaveLength(0);
  });

  it("returns all automations in order", () => {
    const result = getAutomationsForPolicy(
      {
        install_software: { name: "Chrome.app", software_title_id: 1 },
        run_script: { id: 2, name: "fix.sh" },
        calendar_events_enabled: true,
        conditional_access_enabled: true,
        webhook: "On",
      },
      "webhook"
    );
    expect(result.map((a) => a.type)).toEqual([
      "software",
      "script",
      "calendar",
      "conditional_access",
      "other",
    ]);
  });
});

describe("getPageAfterDelete", () => {
  const PAGE_SIZE = 20;

  it("steps back a page when the last row on the last page is deleted", () => {
    // 21 policies, 1 on page 1 (index) -> delete it -> page 1 is now empty.
    expect(
      getPageAfterDelete({
        currentPage: 1,
        totalCount: 21,
        deletedCount: 1,
        pageSize: PAGE_SIZE,
      })
    ).toBe(0);
  });

  it("steps back a page when all rows on a full non-first page are deleted", () => {
    // 40 policies, page 1 is full (20) -> delete all 20 -> page 1 is now empty.
    expect(
      getPageAfterDelete({
        currentPage: 1,
        totalCount: 40,
        deletedCount: 20,
        pageSize: PAGE_SIZE,
      })
    ).toBe(0);
  });

  it("stays on the current page when rows remain after deletion", () => {
    // 40 policies on pages 0 and 1 -> delete 5 from page 1 -> 15 remain there.
    expect(
      getPageAfterDelete({
        currentPage: 1,
        totalCount: 40,
        deletedCount: 5,
        pageSize: PAGE_SIZE,
      })
    ).toBe(1);
  });

  it("never steps back below the first page", () => {
    // Deleting the last policy on page 0 should keep the user on page 0.
    expect(
      getPageAfterDelete({
        currentPage: 0,
        totalCount: 1,
        deletedCount: 1,
        pageSize: PAGE_SIZE,
      })
    ).toBe(0);
  });

  it("steps back from a deeper page when it becomes empty", () => {
    // 61 policies, 1 on page 3 -> delete it -> page 3 empty -> back to page 2.
    expect(
      getPageAfterDelete({
        currentPage: 3,
        totalCount: 61,
        deletedCount: 1,
        pageSize: PAGE_SIZE,
      })
    ).toBe(2);
  });

  it("stays on the current page when the total count is unknown", () => {
    // Count query unresolved/errored -> undefined -> don't guess, stay put.
    expect(
      getPageAfterDelete({
        currentPage: 2,
        totalCount: undefined,
        deletedCount: 1,
        pageSize: PAGE_SIZE,
      })
    ).toBe(2);
  });
});
