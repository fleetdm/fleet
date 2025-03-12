import React from "react";
import {
  getInstallSoftwareErrorMessage,
  getRunScriptErrorMessage,
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
