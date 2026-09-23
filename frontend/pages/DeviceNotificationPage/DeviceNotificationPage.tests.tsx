import { screen, waitFor } from "@testing-library/react";
import React from "react";

import {
  createMockNotificationView,
  customDeviceNotificationHandler,
  defaultDeviceNotificationActionHandler,
  errorDeviceNotificationActionHandler,
  errorDeviceNotificationHandler,
  installingThenSettledDeviceNotificationHandler,
  notFoundDeviceNotificationHandler,
  reminderThenInstallingDeviceNotificationHandler,
  settledDeviceNotificationHandler,
} from "test/handlers/device-notifications-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";

import DeviceNotificationPage from "./DeviceNotificationPage";

const baseClass = "device-notification-page";

const renderPage = () => {
  const render = createCustomRenderer({ withBackendMock: true });
  return render(
    <DeviceNotificationPage
      params={{
        device_auth_token: "test-token",
        notification_uuid: "test-uuid",
      }}
    />
  );
};

describe("DeviceNotificationPage", () => {
  const postMessage = jest.fn();

  beforeEach(() => {
    postMessage.mockClear();
    window.webkit = {
      messageHandlers: { fleetDesktop: { postMessage } },
    };
  });

  afterEach(() => {
    delete window.webkit;
    localStorage.removeItem("fleet-theme");
    // A polling test that fails mid-way would otherwise leave fake timers on
    // and take every test after it down with it.
    jest.useRealTimers();
  });

  it("renders the fetched notification and posts `ready` once", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          title: "Apps will close in 1 hour",
          description: "Save your work.",
          items: [
            {
              software_title_id: 1,
              name: "1Password 8",
              display_name: "1Password",
              icon_url: null,
            },
          ],
          actions: [
            { id: "remind", label: "Remind me in 1 hour" },
            { id: "update_now", label: "Update now" },
          ],
        })
      )
    );

    renderPage();

    expect(
      await screen.findByText("Apps will close in 1 hour")
    ).toBeInTheDocument();
    expect(screen.getByText("Save your work.")).toBeInTheDocument();
    expect(screen.getByText("1Password")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Remind me in 1 hour" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Update now" })
    ).toBeInTheDocument();

    await waitFor(() => {
      expect(postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ action: "ready" })
      );
    });
    const readyCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "ready"
    );
    expect(readyCalls).toHaveLength(1);
  });

  it("posts `error` and renders nothing on a 404", async () => {
    mockServer.use(notFoundDeviceNotificationHandler);

    const { container } = renderPage();

    await waitFor(() => {
      expect(postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ action: "error" })
      );
    });
    expect(container.querySelector(".device-notification-page")).toBeNull();
  });

  it("posts `error` on a 500", async () => {
    mockServer.use(errorDeviceNotificationHandler);

    renderPage();

    await waitFor(() => {
      expect(postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ action: "error" })
      );
    });
  });

  it("does not throw when window.webkit is absent (browser dev)", async () => {
    delete window.webkit;
    mockServer.use(customDeviceNotificationHandler());

    renderPage();

    expect(await screen.findByText(/apps will close/i)).toBeInTheDocument();
  });

  it("renders **bold** markup in title and description as <strong>", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          title: "Apps will close in **1 hour**",
          description: "Save your **work** first.",
        })
      )
    );

    const { container } = renderPage();

    await screen.findByText(/Apps will close in/);
    const strongs = container.querySelectorAll("strong");
    const strongText = Array.from(strongs).map((el) => el.textContent);
    expect(strongText).toContain("1 hour");
    expect(strongText).toContain("work");
  });

  it("renders the last action as the default Button variant and others as subdued", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          actions: [
            { id: "remind", label: "Remind me in 1 hour" },
            { id: "update_now", label: "Update now" },
          ],
        })
      )
    );

    renderPage();

    const primary = await screen.findByRole("button", { name: "Update now" });
    const secondary = screen.getByRole("button", {
      name: "Remind me in 1 hour",
    });
    expect(primary.className).toMatch(/button--default/);
    expect(secondary.className).toMatch(/button--subdued/);
  });

  it("transitions to Installing on update_now and posts no close-triggering bridge", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          actions: [
            { id: "remind", label: "Remind me in 1 hour" },
            { id: "update_now", label: "Update now" },
          ],
        })
      ),
      defaultDeviceNotificationActionHandler
    );

    const { user } = renderPage();

    const primary = await screen.findByRole("button", { name: "Update now" });
    await user.click(primary);

    // Server returns the Updating... view (statuses on every item), which
    // we render by writing the response into the query cache.
    const updating = await screen.findAllByText("Updating...");
    expect(updating.length).toBeGreaterThan(0);

    // Both `primary` and `dismiss` fade the toast out in Swift, so update_now
    // must post neither — the window has to stay open on the Updating view.
    const closingCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "primary" || msg.action === "dismiss"
    );
    expect(closingCalls).toHaveLength(0);
  });

  // An install that is still pending keeps the toast polling, so a status that
  // settles on the server reaches the screen without the end user doing anything.
  it("polls while an install is pending and shows the settled statuses", async () => {
    jest.useFakeTimers();
    const { handler, state } = installingThenSettledDeviceNotificationHandler();
    mockServer.use(handler);

    renderPage();

    expect((await screen.findAllByText("Updating...")).length).toBe(3);
    expect(state.requestCount).toBe(1);

    jest.advanceTimersByTime(5000);

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect((await screen.findAllByText("Updated")).length).toBe(2);
    expect(screen.queryByText("Updating...")).not.toBeInTheDocument();

    // Nothing is pending now, so the interval stops rather than polling forever.
    const settledRequestCount = state.requestCount;
    jest.advanceTimersByTime(15000);
    await waitFor(() => {
      expect(state.requestCount).toBe(settledRequestCount);
    });
  });

  it("a 5 minute reminder left open refetches at install_at and every minute after, and shows its queued installs as Updating...", async () => {
    jest.useFakeTimers();
    const {
      handler,
      state,
    } = reminderThenInstallingDeviceNotificationHandler();
    mockServer.use(handler);

    renderPage();

    expect(
      await screen.findByRole("button", { name: "Update now" })
    ).toBeInTheDocument();
    expect(state.requestCount).toBe(1);

    // skip refetching the reminder toast while its install_at is still ahead
    jest.advanceTimersByTime(64000);
    await waitFor(() => {
      expect(state.requestCount).toBe(1);
    });

    // refetch the reminder toast at install_at and show no status because the installs are not queued
    jest.advanceTimersByTime(1000);
    await waitFor(() => {
      expect(state.requestCount).toBe(2);
    });
    expect(screen.queryByText("Updating...")).not.toBeInTheDocument();

    // skip refetching the reminder toast for the rest of the minute after install_at
    jest.advanceTimersByTime(59000);
    await waitFor(() => {
      expect(state.requestCount).toBe(2);
    });

    // refetch the reminder toast a minute after install_at and show the queued installs as Updating...
    jest.advanceTimersByTime(1000);
    expect((await screen.findAllByText("Updating...")).length).toBe(3);
    expect(state.requestCount).toBe(3);
  });

  // The toast stays open on a terminal view: the end user closes it with Hide.
  it("keeps the toast open and offers Hide once every install has settled", async () => {
    mockServer.use(settledDeviceNotificationHandler);

    renderPage();

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Hide" })).toBeInTheDocument();
    const dismissCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "dismiss"
    );
    expect(dismissCalls).toHaveLength(0);
  });

  it("posts `dismiss` bridge on a secondary action", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          actions: [
            { id: "remind", label: "Remind me in 1 hour" },
            { id: "update_now", label: "Update now" },
          ],
        })
      ),
      defaultDeviceNotificationActionHandler
    );

    const { user } = renderPage();

    const secondary = await screen.findByRole("button", {
      name: "Remind me in 1 hour",
    });
    await user.click(secondary);

    await waitFor(() => {
      expect(postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ action: "dismiss" })
      );
    });
    // Secondary is not the primary — no `primary` bridge.
    const primaryCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "primary"
    );
    expect(primaryCalls).toHaveLength(0);
  });

  it("posts `dismiss` (not `primary`) when the sole action's id is `dismiss` — Installing-state Hide", async () => {
    // Post-`update_now` view: single Hide action whose id is `dismiss`. It is
    // positionally the primary (last of 1) but semantically closes the toast.
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          actions: [{ id: "dismiss", label: "Hide" }],
        })
      ),
      defaultDeviceNotificationActionHandler
    );

    const { user } = renderPage();

    const hide = await screen.findByRole("button", { name: "Hide" });
    await user.click(hide);

    await waitFor(() => {
      expect(postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ action: "dismiss" })
      );
    });
    const primaryCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "primary"
    );
    expect(primaryCalls).toHaveLength(0);
  });

  it("surfaces an inline error when the action POST fails", async () => {
    mockServer.use(
      customDeviceNotificationHandler(),
      errorDeviceNotificationActionHandler
    );

    const { user } = renderPage();

    const primary = await screen.findByRole("button", { name: "Update now" });
    await user.click(primary);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /please try again/i
    );
    // A failed POST must not silently send the outcome bridge messages.
    const primaryCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "primary"
    );
    const dismissCalls = postMessage.mock.calls.filter(
      ([msg]) => msg.action === "dismiss"
    );
    expect(primaryCalls).toHaveLength(0);
    expect(dismissCalls).toHaveLength(0);
  });

  it("renders the light-mode logo in light mode and the dark-mode logo in dark mode", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          org_logo_url_light_mode: "https://example.com/light.png",
          org_logo_url_dark_mode: "https://example.com/dark.png",
        })
      )
    );

    const light = renderPage();

    await waitFor(() => {
      const src = light.container
        .querySelector(`.${baseClass}__logo`)
        ?.getAttribute("src");
      expect(src).toBe("https://example.com/light.png");
    });
    light.unmount();

    localStorage.setItem("fleet-theme", "dark");
    const dark = renderPage();

    await waitFor(() => {
      const src = dark.container
        .querySelector(`.${baseClass}__logo`)
        ?.getAttribute("src");
      expect(src).toBe("https://example.com/dark.png");
    });
  });

  // An org that never uploaded a logo gets empty urls from the server, and OrgLogoIcon
  // substitutes Fleet's own logo rather than rendering a broken image.
  it("renders Fleet's logo when the org has not set one", async () => {
    mockServer.use(
      customDeviceNotificationHandler(
        createMockNotificationView({
          org_logo_url_light_mode: "",
          org_logo_url_dark_mode: "",
        })
      )
    );

    const { container } = renderPage();

    await waitFor(() => {
      const logo = container.querySelector(`.${baseClass}__logo`);
      expect(logo).toHaveClass("default-fleet-logo");
    });
  });
});
