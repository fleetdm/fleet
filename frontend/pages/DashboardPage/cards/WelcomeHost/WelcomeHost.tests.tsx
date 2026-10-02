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
});
