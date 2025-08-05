import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { UserEvent } from "@testing-library/user-event";
import { IScript } from "interfaces/script";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import RunScriptBatchModal from "./RunScriptBatchModal";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const windowsScript: IScript = {
  id: 123,
  team_id: 1,
  name: "winscript.ps1",
  created_at: "2023-01-01T00:00:00Z",
  updated_at: "2023-01-01T00:00:00Z",
};

const linuxScript: IScript = {
  id: 456,
  team_id: 1,
  name: "linuxscript.sh",
  created_at: "2023-01-01T00:00:00Z",
  updated_at: "2023-01-01T00:00:00Z",
};

jest.mock("../RunScriptBatchPaginatedList", () => {
  return {
    __esModule: true,
    default: ({ onRunScript }: { onRunScript: (script: IScript) => void }) => {
      return (
        <div>
          <div onClick={() => onRunScript(windowsScript)}>windows script</div>
          <div onClick={() => onRunScript(linuxScript)}>linux script</div>
        </div>
      );
    },
  };
});

const selectScript = async (user: UserEvent, platform: string) => {
  let el;
  await waitFor(async () => {
    el = screen.getByText(`${platform} script`);
    expect(el).toBeInTheDocument();
  });
  if (!el) {
    throw new Error("Script element not found");
  }
  await user.click(el);
  let runButton;
  let cancelButton;
  await waitFor(() => {
    runButton = screen.getByRole("button", { name: "Run" });
    expect(runButton).toBeInTheDocument();
    cancelButton = screen.getByRole("button", { name: "Cancel" });
    expect(cancelButton).toBeInTheDocument();
  });
  if (!runButton || !cancelButton) {
    throw new Error("Run or Cancel button not found");
  }
  return { runButton, cancelButton };
};

const getScheduleSelector = async () => {
  let scheduleButton;
  let runNowButton;
  await waitFor(() => {
    scheduleButton = screen.getByLabelText("Schedule for later");
    expect(scheduleButton).toBeInTheDocument();
    runNowButton = screen.getByLabelText("Run now");
    expect(runNowButton).toBeInTheDocument();
  });
  if (!scheduleButton || !runNowButton) {
    throw new Error("Schedule or Run Now button not found");
  }
  return { scheduleButton, runNowButton };
};

const getScheduleUI = async () => {
  let dateInput;
  let timeInput;
  await waitFor(() => {
    dateInput = screen.getByLabelText("Date");
    expect(dateInput).toBeInTheDocument();
    timeInput = screen.getByLabelText("Time");
    expect(timeInput).toBeInTheDocument();
  });
  if (!dateInput || !timeInput) {
    throw new Error("Date or Time input not found");
  }
  return { dateInput, timeInput };
};

describe("RunScriptBatchModal", () => {
  const scriptsHandler = http.get(baseUrl("/scripts"), () => {
    return HttpResponse.json({
      scripts: [windowsScript, linuxScript],
    });
  });

  const runBatchFn = jest.fn(async (req) => {
    return HttpResponse.json({});
  });
  const runBatchHandler = http.post(baseUrl("/scripts/run/batch"), runBatchFn);

  beforeEach(() => {
    mockServer.use(scriptsHandler);
    runBatchFn.mockReset();
    mockServer.use(runBatchHandler);
  });

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  const defaultProps = {
    runByFilters: false,
    // since teamId has multiple uses in this component, it's passed in as its own prop and added to
    // `filters` as needed
    filters: { team_id: 1, status: "" },
    teamId: 1,
    // If we are on the free tier, we don't want to apply any kind of team filters (since the feature is Premium only).
    isFreeTier: false,
    totalFilteredHostsCount: 2,
    selectedHostIds: [1, 2],
    onCancel: () => null,
  };

  it("lists the scripts available for batch running", () => {
    render(<RunScriptBatchModal {...defaultProps} />);
    waitFor(() => {
      const windowsScriptElement = screen.getByText("windows script");
      const linuxScriptElement = screen.getByText("linux script");
      expect(windowsScriptElement).toBeInTheDocument();
      expect(linuxScriptElement).toBeInTheDocument();
    });
  });

  describe("after clicking run script", () => {
    it("shows the correct heading for linux/macos scripts", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);
      await selectScript(user, "linux");

      await waitFor(() => {
        expect(screen.getByText("linuxscript.sh")).toBeInTheDocument();
        expect(screen.getByText(/macOS\/linux/)).toBeInTheDocument();
      });
    });

    it("shows the correct heading for windows", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);
      await selectScript(user, "windows");

      await waitFor(() => {
        expect(screen.getByText("winscript.ps1")).toBeInTheDocument();
        expect(screen.getByText(/windows/)).toBeInTheDocument();
      });
    });

    it("does not show the scheduling UI if 'run now' is selected", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);
      await selectScript(user, "windows");
      const { runNowButton } = await getScheduleSelector();
      expect(runNowButton).toBeChecked();
      expect(screen.queryByLabelText(/Date/)).not.toBeInTheDocument();
      expect(screen.queryByLabelText(/Time/)).not.toBeInTheDocument();
    });

    it("shows the scheduling UI if 'schedule for later' is selected", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);
      await selectScript(user, "windows");
      const { runNowButton, scheduleButton } = await getScheduleSelector();
      expect(runNowButton).toBeChecked();
      await user.click(scheduleButton);
      await getScheduleUI();
    });

    describe("run now", () => {
      it("should call the API with no not_before param", async () => {
        const { user } = render(<RunScriptBatchModal {...defaultProps} />);
        const { runButton } = await selectScript(user, "windows");
        await user.click(runButton);
        expect(runBatchFn.mock.calls.length).toBe(1);
        const body = await runBatchFn.mock.calls[0][0].request.json();
        expect(body).toEqual({
          script_id: windowsScript.id,
          host_ids: defaultProps.selectedHostIds,
        });
      });
    });

    describe("schedule for later", () => {
      it("requires a valid date", async () => {
        const { user } = render(<RunScriptBatchModal {...defaultProps} />);
        await selectScript(user, "windows");
        const { runNowButton, scheduleButton } = await getScheduleSelector();
        expect(runNowButton).toBeChecked();
        await user.click(scheduleButton);
        const { dateInput } = await getScheduleUI();
        // Add a wildly invalid date
        await user.type(dateInput, "u up?");
        expect(dateInput).toHaveValue("u up?");
        expect(
          screen.getByText("Please enter a valid date.")
        ).toBeInTheDocument();
      });

      it("requires a valid time", async () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });

      it("should call the API with a not_before param", async () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });
    });
  });
});
