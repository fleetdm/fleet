import React from "react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import {
  getInstallSoftwareErrorMessage,
  getRunScriptErrorMessage,
  getAutomationTypesString,
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

describe("getAutomationTypesString", () => {
  const basePolicy = {
    calendar_events_enabled: false,
    conditional_access_enabled: false,
  };

  it("returns DEFAULT_EMPTY_CELL_VALUE when no automations are enabled", () => {
    expect(getAutomationTypesString(basePolicy)).toBe(DEFAULT_EMPTY_CELL_VALUE);
  });

  it("returns 'Software' when only install_software is present", () => {
    expect(
      getAutomationTypesString({
        ...basePolicy,
        install_software: { software_title_id: 1 },
      })
    ).toBe("Software");
  });

  it("returns 'Script' when only run_script is present", () => {
    expect(
      getAutomationTypesString({
        ...basePolicy,
        run_script: { id: 1 },
      })
    ).toBe("Script");
  });

  it("returns types in correct order with sentence case: Software, script, calendar, conditional access, other", () => {
    expect(
      getAutomationTypesString({
        install_software: { software_title_id: 1 },
        run_script: { id: 1 },
        calendar_events_enabled: true,
        conditional_access_enabled: true,
        webhook: "On",
      })
    ).toBe("Software, script, calendar, conditional access, other");
  });

  it("returns 'Software, calendar' for software + calendar", () => {
    expect(
      getAutomationTypesString({
        ...basePolicy,
        install_software: { software_title_id: 1 },
        calendar_events_enabled: true,
      })
    ).toBe("Software, calendar");
  });

  it("does not include Other when webhook is Off", () => {
    expect(
      getAutomationTypesString({
        ...basePolicy,
        install_software: { software_title_id: 1 },
        webhook: "Off",
      })
    ).toBe("Software");
  });
});
