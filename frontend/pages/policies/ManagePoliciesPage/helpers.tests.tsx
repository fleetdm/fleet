import React from "react";
import { getErrorMessage } from "./helpers";
import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";

describe("getErrorMessage", () => {
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
                "software_title_id 178 on team_id 789 does not have associated package",
            },
          ],
        },
      },
      status: "rejected",
    };

    const currentTeamName = "1a - Workstations (canary)";

    const result = getErrorMessage(mockResult, mockFormData, currentTeamName);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Keynote.app (ID: 178) on 1a - Workstations (canary) does not have associated package"
    );
  });

  it("handles unknown software_title_id", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with software_title_id 999",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with software_title_id 999"
    );
  });

  it("handles missing currentTeamName", () => {
    const mockResult: PromiseRejectedResult = {
      reason: {
        data: {
          errors: [
            {
              reason: "Error with team_id 789",
            },
          ],
        },
      },
      status: "rejected",
    };

    const result = getErrorMessage(mockResult, mockFormData);
    const resultString = renderToString(result);

    expect(resultString).toBe(
      "Could not update policy. Error with team_id 789"
    );
  });
});
