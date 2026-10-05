import { act, screen, waitFor } from "@testing-library/react";
import React from "react";

import createMockHost from "__mocks__/hostMock";
import { notify } from "components/ToastNotification";
import { IHost } from "interfaces/host";
import { IHostPolicy } from "interfaces/policy";
import hostAPI from "services/entities/hosts";
import { createCustomRenderer } from "test/test-utils";

import WelcomeHost from "./WelcomeHost";

jest.mock("services/entities/hosts");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const PASSING_POLICY: IHostPolicy = {
  id: 1,
  name: "Antivirus healthy",
  query: "SELECT 1",
  description: "test",
  author_id: 1,
  author_name: "Test User",
  author_email: "test@example.com",
  resolution: "",
  platform: "darwin",
  team_id: null,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-01T00:00:00Z",
  critical: false,
  calendar_events_enabled: false,
  conditional_access_enabled: false,
  type: "dynamic",
  response: "pass",
};

const mockOnlineStuckRefetchHost = (): IHost =>
  createMockHost({
    id: 1,
    status: "online",
    refetch_requested: true,
    policies: [PASSING_POLICY],
    detail_updated_at: "2024-01-01T00:00:00Z",
  });

const renderWelcomeHost = () => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(
    <WelcomeHost totalsHostsCount={1} toggleAddHostsModal={jest.fn()} />
  );
};

describe("WelcomeHost - refetch give-up state", () => {
  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
    jest.resetAllMocks();
  });

  it("resets the refetch timer on give-up so a later, unrelated host re-fetch doesn't immediately re-show the toast", async () => {
    jest.useFakeTimers({ doNotFake: ["queueMicrotask"] });

    let now = 1_700_000_000_000;
    jest.spyOn(Date, "now").mockImplementation(() => now);

    (hostAPI.loadHostDetails as jest.Mock)
      // 1) initial load: starts the refetch timer and schedules a 1s poll
      .mockResolvedValueOnce({ host: mockOnlineStuckRefetchHost() })
      // 2) the 1s poll's result: by the time it arrives, 61s have "passed"
      .mockResolvedValueOnce({ host: mockOnlineStuckRefetchHost() })
      // 3) a later, unrelated re-fetch (e.g. refetchOnWindowFocus): still stuck
      .mockResolvedValue({ host: mockOnlineStuckRefetchHost() });

    renderWelcomeHost();
    await screen.findByText("Antivirus healthy");

    // Let the component's 1s poll fire. Advance the mocked clock past the
    // 60s give-up threshold first, so the *next* onSuccess hits that branch.
    now += 61000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(1000);
    });

    await waitFor(() => {
      expect(hostAPI.loadHostDetails).toHaveBeenCalledTimes(2);
    });
    expect(notify.error).toHaveBeenCalledTimes(1);
    expect(notify.error).toHaveBeenCalledWith(
      "Refetch sent but vitals are taking longer than expected to load. You’ll see an update when the host responds."
    );

    // Simulate window focus regaining, the exact trigger from the issue's
    // repro steps (react-query's refetchOnWindowFocus). If refetchStartTime
    // wasn't reset on give-up, this alone re-triggers the give-up branch
    // immediately, since `Date.now() - <stale timestamp>` is still >= 60000.
    await act(async () => {
      window.dispatchEvent(new Event("visibilitychange"));
      window.dispatchEvent(new Event("focus"));
      await jest.advanceTimersByTimeAsync(0);
    });

    await waitFor(() => {
      expect(hostAPI.loadHostDetails).toHaveBeenCalledTimes(3);
    });

    // Still only the one call from the original give-up -- not a second one.
    expect(notify.error).toHaveBeenCalledTimes(1);
  });

  // #54676: each tab-focus re-entry into onSuccess used to schedule a fresh
  // setTimeout next to the one already pending, so polling sped up.
  it("does not stack polling loops when the tab regains focus mid-refetch", async () => {
    jest.useFakeTimers({ doNotFake: ["queueMicrotask"] });
    let now = 1_700_000_000_000;
    jest.spyOn(Date, "now").mockImplementation(() => now);

    (hostAPI.loadHostDetails as jest.Mock).mockResolvedValue({
      host: mockOnlineStuckRefetchHost(),
    });

    renderWelcomeHost();
    await screen.findByText("Antivirus healthy");

    for (let i = 0; i < 3; i += 1) {
      // eslint-disable-next-line no-await-in-loop
      await act(async () => {
        window.dispatchEvent(new Event("focus"));
        window.dispatchEvent(new Event("visibilitychange"));
        await jest.advanceTimersByTimeAsync(0);
      });
    }

    const callsBeforeAdvance = (hostAPI.loadHostDetails as jest.Mock).mock.calls
      .length;

    now += 1000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(1000);
    });

    const pollsInOneInterval =
      (hostAPI.loadHostDetails as jest.Mock).mock.calls.length -
      callsBeforeAdvance;
    expect(pollsInOneInterval).toBe(1);
  });

  // #54677: after give-up, a focus-triggered onSuccess used to re-enter the
  // "timer just started" branch, open a fresh 60s cycle, and fire the toast
  // again 60s later.
  it("doesn't restart the refetch window after a timeout when the tab regains focus", async () => {
    jest.useFakeTimers({ doNotFake: ["queueMicrotask"] });
    let now = 1_700_000_000_000;
    jest.spyOn(Date, "now").mockImplementation(() => now);

    (hostAPI.loadHostDetails as jest.Mock).mockResolvedValue({
      host: mockOnlineStuckRefetchHost(),
    });

    renderWelcomeHost();
    await screen.findByText("Antivirus healthy");

    // Trip the give-up branch on the first polling tick.
    now += 61000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(1000);
    });

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledTimes(1);
    });

    // Tab switch, then let a full fresh 60s window elapse. Without the fix,
    // the focus-triggered onSuccess restarts a new cycle and the toast fires
    // a second time around now.
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
      window.dispatchEvent(new Event("visibilitychange"));
      await jest.advanceTimersByTimeAsync(0);
    });
    now += 65000;
    await act(async () => {
      await jest.advanceTimersByTimeAsync(1000);
    });

    expect(notify.error).toHaveBeenCalledTimes(1);
  });
});
