import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { noop } from "lodash";

import { baseUrl, createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { IActivityDetails } from "interfaces/activity";
import { IScriptResultResponse } from "services/entities/scripts";

import NotifyBeforePatchingDetailsModal from "./NotifyBeforePatchingDetailsModal";

const scriptResult = (
  overrides: Partial<IScriptResultResponse> = {}
): IScriptResultResponse => ({
  hostname: "John's MacBook Pro",
  host_id: 1,
  execution_id: "exec-1",
  script_contents: "",
  script_id: null,
  exit_code: 0,
  output: "",
  message: "",
  runtime: 1,
  host_timeout: false,
  created_at: "2026-01-01T00:00:00Z",
  ...overrides,
});

const useScriptResultHandler = (overrides?: Partial<IScriptResultResponse>) => {
  mockServer.use(
    http.get(baseUrl("/scripts/results/:executionId"), () =>
      HttpResponse.json(scriptResult(overrides))
    )
  );
};

const defaultDetails: IActivityDetails = {
  host_display_name: "John's MacBook Pro",
  software_titles: ["1Password"],
  status: "success",
  time_before: 3600,
  script_execution_id: "exec-1",
};

const renderModal = (details: Partial<IActivityDetails> = {}) => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(
    <NotifyBeforePatchingDetailsModal
      details={{ ...defaultDetails, ...details }}
      onCancel={noop}
    />
  );
};

describe("NotifyBeforePatchingDetailsModal", () => {
  it("renders the success sentence with 1 hour by default", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({ status: "success", time_before: 3600 });

    expect(
      await screen.findByText(/notified end user 1 hour before patching/i)
    ).toBeInTheDocument();
    expect(screen.getByText("John's MacBook Pro")).toBeInTheDocument();
  });

  it("renders 5 minutes for the reminder", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({ status: "success", time_before: 300 });

    expect(
      await screen.findByText(/notified end user 5 minutes before patching/i)
    ).toBeInTheDocument();
  });

  it("renders the offline sentence for a success with exit code 0", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(/If the host is offline when the patch is forced/)
    ).toBeInTheDocument();
  });

  it("renders the Fleet Desktop required sentence for exit code 100", async () => {
    useScriptResultHandler({ exit_code: 100 });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(
        /The Fleet Desktop app is required to notify end users\./
      )
    ).toBeInTheDocument();
  });

  it("renders the Fleet Desktop v1.5.0 sentence for exit code 101", async () => {
    useScriptResultHandler({ exit_code: 101 });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(/The Fleet Desktop app v1\.5\.0 is required/)
    ).toBeInTheDocument();
  });

  it("renders the 'notification couldn't load' sentence for exit code 30", async () => {
    useScriptResultHandler({ exit_code: 30 });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(/The notification couldn't load\./)
    ).toBeInTheDocument();
  });

  it("renders the screen-locked sentence for exit code 41", async () => {
    useScriptResultHandler({ exit_code: 41 });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(/screen was locked so the end user couldn't see/)
    ).toBeInTheDocument();
  });

  it("renders output only (no failure sentence) for an unknown exit code", async () => {
    useScriptResultHandler({ exit_code: 999, output: "unknown error output" });
    renderModal({ status: "failed" });

    expect(await screen.findByText(/unknown error output/)).toBeInTheDocument();
    // None of the documented failure sentences should appear.
    expect(
      screen.queryByText(/The Fleet Desktop app is required/)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/The notification couldn't load/)
    ).not.toBeInTheDocument();
  });

  it("renders the deferred sentence when script_execution_id is absent, no fetch fired", async () => {
    // Handler set to fail loudly if hit — absence of execution id must skip
    // the fetch entirely.
    const shouldNotFireHandler = jest.fn();
    mockServer.use(
      http.get(baseUrl("/scripts/results/:executionId"), () => {
        shouldNotFireHandler();
        return HttpResponse.json(scriptResult());
      })
    );

    renderModal({ status: "failed", script_execution_id: undefined });

    expect(
      await screen.findByText(
        /Another notification was displayed\. Fleet will try again on the next policy run\./
      )
    ).toBeInTheDocument();
    expect(shouldNotFireHandler).not.toHaveBeenCalled();
  });

  it("lists every title in the Apps row (no truncation past three)", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({
      software_titles: [
        "1Password",
        "Slack",
        "Docker Desktop",
        "Zoom",
        "Chrome",
      ],
      status: "success",
    });

    const appsRow = await screen.findByText("Apps");
    const appsSection = appsRow.parentElement;
    expect(appsSection?.textContent).toContain("1Password");
    expect(appsSection?.textContent).toContain("Slack");
    expect(appsSection?.textContent).toContain("Docker Desktop");
    expect(appsSection?.textContent).toContain("Zoom");
    expect(appsSection?.textContent).toContain("Chrome");
    // No truncation suffix.
    expect(appsSection?.textContent).not.toMatch(/more app/);
  });

  it("renders the notification script output when present", async () => {
    useScriptResultHandler({
      exit_code: 41,
      output: "Screen locked at 2026-08-18T10:00:00Z",
    });
    renderModal({ status: "failed" });

    expect(
      await screen.findByText(/Notification script output:/)
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByText(/Screen locked at 2026-08-18T10:00:00Z/)
      ).toBeInTheDocument()
    );
  });
});
