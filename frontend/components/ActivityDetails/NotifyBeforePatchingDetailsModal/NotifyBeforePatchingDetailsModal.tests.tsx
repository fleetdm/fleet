import React from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { noop } from "lodash";

import { baseUrl, createCustomRenderer } from "test/test-utils";
import mockServer from "test/mock-server";
import { IActivityDetails } from "interfaces/activity";
import { IScriptResultResponse } from "services/entities/scripts";

import NotifyBeforePatchingDetailsModal from "./NotifyBeforePatchingDetailsModal";

// Default carries a stub script body so the Details reveal has something to
// show — matches real payload shape where notification runs always have
// contents.
const DEFAULT_SCRIPT_CONTENTS = "#!/bin/bash\nfleet-desktop notify ...";

const scriptResult = (
  overrides: Partial<IScriptResultResponse> = {}
): IScriptResultResponse => ({
  hostname: "John's MacBook Pro",
  host_id: 1,
  execution_id: "exec-1",
  script_contents: DEFAULT_SCRIPT_CONTENTS,
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

  it("renders the offline caveat sentence on a success (exit code 0)", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({ status: "success" });

    expect(
      await screen.findByText(/If the host is offline when the patch is forced/)
    ).toBeInTheDocument();
    // Intro is the success verb, not the failure verb.
    expect(
      screen.getByText(/notified end user 1 hour before patching/i)
    ).toBeInTheDocument();
    expect(screen.queryByText(/failed to notify/)).not.toBeInTheDocument();
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

    // Reveal to expose the raw output.
    await userEvent.click(
      await screen.findByRole("button", { name: /Details/ })
    );
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

  it("puts the app name inline in the intro when there is one app, and hides the Apps row", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({ status: "success", software_titles: ["1Password"] });

    // Intro carries the app name.
    expect(await screen.findByText(/before patching/i)).toBeInTheDocument();
    const intro = screen
      .getByText(/before patching/i)
      .closest("span") as HTMLElement;
    expect(intro.textContent).toContain("1Password");
    expect(intro.textContent).toContain("John's MacBook Pro");

    // Apps row is redundant for a single-app case — omitted.
    expect(screen.queryByText("Apps")).not.toBeInTheDocument();
  });

  it("truncates the intro past three apps with a 'and N more apps' tail and moves the full list to Apps", async () => {
    useScriptResultHandler({ exit_code: 0 });
    renderModal({
      status: "success",
      software_titles: [
        "1Password",
        "Slack",
        "Docker Desktop",
        "Zoom",
        "Chrome",
      ],
    });

    const intro = (await screen.findByText(/before patching/i)).closest(
      "span"
    ) as HTMLElement;
    expect(intro.textContent).toMatch(/and 2 more apps/);
    // Only the first three are named in the intro.
    expect(intro.textContent).toContain("1Password");
    expect(intro.textContent).toContain("Slack");
    expect(intro.textContent).toContain("Docker Desktop");
    expect(intro.textContent).not.toContain("Zoom");
    expect(intro.textContent).not.toContain("Chrome");

    // Apps row appears because the intro truncated.
    expect(screen.getByText("Apps")).toBeInTheDocument();
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

  it("hides the script + output behind a Details reveal", async () => {
    useScriptResultHandler({
      exit_code: 41,
      output: "Screen locked at 2026-08-18T10:00:00Z",
      script_contents: "#!/bin/bash\nnotify ...",
    });
    renderModal({ status: "failed" });

    // Nothing revealed initially.
    await screen.findByText(/screen was locked/i);
    expect(screen.queryByText(/Notification script:/)).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /Details/ }));

    expect(screen.getByText(/Notification script:/)).toBeInTheDocument();
    expect(screen.getByText(/#!\/bin\/bash/)).toBeInTheDocument();
    // The label is split around the "output recorded" tooltip trigger, so
    // match the trigger substring rather than the whole sentence.
    expect(screen.getByText("output recorded")).toBeInTheDocument();
    // Output block leads with the exit code line per Figma.
    expect(screen.getByText(/Exit code: 41/)).toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByText(/Screen locked at 2026-08-18T10:00:00Z/)
      ).toBeInTheDocument()
    );
  });

  it("shows a green success icon on success and a red error icon on failure", async () => {
    useScriptResultHandler({ exit_code: 0 });
    const { unmount } = renderModal({ status: "success" });
    expect(
      await screen.findByTestId("success-outline-icon")
    ).toBeInTheDocument();
    unmount();

    useScriptResultHandler({ exit_code: 41 });
    renderModal({ status: "failed" });
    expect(await screen.findByTestId("error-outline-icon")).toBeInTheDocument();
  });

  it("does not show the Details reveal when no script ran (dispatcher deferral)", async () => {
    renderModal({ status: "failed", script_execution_id: undefined });

    await screen.findByText(/Another notification was displayed/);
    // No script means nothing to reveal.
    expect(
      screen.queryByRole("button", { name: /Details/ })
    ).not.toBeInTheDocument();
  });
});
