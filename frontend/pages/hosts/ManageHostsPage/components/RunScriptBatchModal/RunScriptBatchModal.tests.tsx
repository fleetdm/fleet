import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { UserEvent } from "@testing-library/user-event";
import { IScript } from "interfaces/script";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import { format } from "date-fns";

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

// Utility to validate that the script list is rendered, select a script,
// and return the run and cancel buttons.
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

// Utility to validate that the "schedule for later" and "run now" buttons are present
// and return them.
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

// Utility to validate that the scheduling UI is present and return the date and time inputs.
const getScheduleUI = async () => {
  let dateInput;
  let timeInput;
  await waitFor(() => {
    dateInput = screen.getByLabelText("Date (UTC)");
    expect(dateInput).toBeInTheDocument();
    timeInput = screen.getByLabelText("Time (UTC)");
    expect(timeInput).toBeInTheDocument();
  });
  if (!dateInput || !timeInput) {
    throw new Error("Date or Time input not found");
  }
  return { dateInput, timeInput };
};

describe("RunScriptBatchModal", () => {
  // Mock the scripts endpoint to return our two test scripts.
  const scriptsHandler = http.get(baseUrl("/scripts"), () => {
    return HttpResponse.json({
      scripts: [windowsScript, linuxScript],
    });
  });

  // Mock the run batch endpoint to simulate running a batch script,
  // and provide a mock function we can use to validate the API call.
  const runBatchFn = jest.fn(async () => {
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

  // Standard props to use for most tests.
  const defaultProps = {
    runByFilters: false,
    filters: { team_id: 1, status: "" },
    teamId: 1,
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
        // @ts-ignore
        const body = await runBatchFn.mock.calls[0][0].request.json();
        expect(body).toEqual({
          script_id: windowsScript.id,
          host_ids: defaultProps.selectedHostIds,
        });
      });

      it("should call the API with filters if supplied", async () => {
        const props = {
          ...defaultProps,
          runByFilters: true,
          filters: { query: "hi", label_id: 16, status: "" },
        };
        props.selectedHostIds = [];
        const { user } = render(<RunScriptBatchModal {...props} />);
        const { runButton } = await selectScript(user, "windows");
        await user.click(runButton);
        expect(runBatchFn.mock.calls.length).toBe(1);
        // @ts-ignore
        const body = await runBatchFn.mock.calls[0][0].request.json();
        expect(body).toEqual({
          script_id: windowsScript.id,
          filters: {
            query: "hi",
            label_id: 16,
            team_id: 1,
            status: "",
          },
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
        // Add a less wild, but still invalid date
        await user.clear(dateInput);
        await user.type(dateInput, "2023-99-99");
        expect(dateInput).toHaveValue("2023-99-99");
        expect(
          screen.getByText("Please enter a valid date.")
        ).toBeInTheDocument();
        // Add a valid date, but in the past
        await user.clear(dateInput);
        await user.type(dateInput, "2023-01-01");
        expect(dateInput).toHaveValue("2023-01-01");
        expect(
          screen.getByText("Date cannot be in the past.")
        ).toBeInTheDocument();
        // Add a valid date in the future
        await user.clear(dateInput);
        await user.type(dateInput, "2099-12-31");
        expect(dateInput).toHaveValue("2099-12-31");
        expect(
          screen.queryByText("Please enter a valid date.")
        ).not.toBeInTheDocument();
        expect(
          screen.queryByText("Date cannot be in the past.")
        ).not.toBeInTheDocument();
      });

      it("requires a valid time", async () => {
        const { user } = render(<RunScriptBatchModal {...defaultProps} />);
        await selectScript(user, "windows");
        const { runNowButton, scheduleButton } = await getScheduleSelector();
        expect(runNowButton).toBeChecked();
        await user.click(scheduleButton);
        const { dateInput, timeInput } = await getScheduleUI();
        // Add a wildly invalid time
        await user.type(timeInput, "professor churro");
        expect(timeInput).toHaveValue("professor churro");
        expect(
          screen.getByText("Please enter a valid time.")
        ).toBeInTheDocument();
        // Add a less wild, but still invalid time
        await user.clear(timeInput);
        await user.type(timeInput, "99:99");
        expect(timeInput).toHaveValue("99:99");
        expect(
          screen.getByText("Please enter a valid time.")
        ).toBeInTheDocument();
        // Add a valid time in the past (no date selected)
        await user.clear(timeInput);
        await user.type(timeInput, "00:00");
        expect(timeInput).toHaveValue("00:00");
        expect(
          screen.queryByText("Please enter a valid date.")
        ).not.toBeInTheDocument();
        expect(
          screen.queryByText("Date cannot be in the past.")
        ).not.toBeInTheDocument();
        // Add a valid time in the past (future date selected)
        await user.clear(timeInput);
        await user.type(timeInput, "00:00");
        await user.clear(dateInput);
        await user.type(dateInput, "2099-12-31");
        expect(timeInput).toHaveValue("00:00");
        expect(
          screen.queryByText("Please enter a valid date.")
        ).not.toBeInTheDocument();
        expect(
          screen.queryByText("Date cannot be in the past.")
        ).not.toBeInTheDocument();
        // Add a valid time in the past (today selected)
        await user.clear(dateInput);
        await user.type(dateInput, format(new Date(), "yyyy-MM-dd"));
        await user.clear(timeInput);
        await user.type(timeInput, "00:00");
        expect(timeInput).toHaveValue("00:00");
        expect(
          screen.getByText("Time cannot be in the past.")
        ).toBeInTheDocument();
      });

      it("should call the API with a correct not_before param", async () => {
        const { user } = render(<RunScriptBatchModal {...defaultProps} />);
        const { runButton } = await selectScript(user, "windows");
        const { runNowButton, scheduleButton } = await getScheduleSelector();
        expect(runNowButton).toBeChecked();
        await user.click(scheduleButton);
        expect(scheduleButton).toBeChecked();
        const { dateInput, timeInput } = await getScheduleUI();
        await user.type(dateInput, "2099-12-31");
        await user.type(timeInput, "23:59");
        await user.click(runButton);
        expect(runBatchFn.mock.calls.length).toBe(1);
        // @ts-ignore
        const body = await runBatchFn.mock.calls[0][0].request.json();
        expect(body).toEqual({
          script_id: windowsScript.id,
          host_ids: defaultProps.selectedHostIds,
          not_before: "2099-12-31 23:59:00.000Z",
        });
      });

      it("should call the API with a correct not_before param and filters if provided", async () => {
        const props = {
          ...defaultProps,
          runByFilters: true,
          filters: { query: "hi", label_id: 16, status: "" },
        };
        props.selectedHostIds = [];
        const { user } = render(<RunScriptBatchModal {...props} />);
        const { runButton } = await selectScript(user, "windows");
        const { runNowButton, scheduleButton } = await getScheduleSelector();
        expect(runNowButton).toBeChecked();
        await user.click(scheduleButton);
        expect(scheduleButton).toBeChecked();
        const { dateInput, timeInput } = await getScheduleUI();
        await user.type(dateInput, "2099-12-31");
        await user.type(timeInput, "23:59");
        await user.click(runButton);
        expect(runBatchFn.mock.calls.length).toBe(1);
        // @ts-ignore
        const body = await runBatchFn.mock.calls[0][0].request.json();
        expect(body).toEqual({
          script_id: windowsScript.id,
          not_before: "2099-12-31 23:59:00.000Z",
          filters: { query: "hi", label_id: 16, status: "", team_id: 1 },
        });
      });
    });
  });
});
