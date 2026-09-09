import React from "react";
import { screen, waitFor } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockNotificationView,
  customDeviceNotificationHandler,
  defaultDeviceNotificationActionHandler,
  errorDeviceNotificationActionHandler,
  errorDeviceNotificationHandler,
  notFoundDeviceNotificationHandler,
} from "test/handlers/device-notifications-handlers";

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

  it("posts `primary` bridge and transitions to Installing on the primary action", async () => {
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

    // Server returns the Installing… view (statuses on every item), which
    // we render by writing the response into the query cache.
    const installing = await screen.findAllByText("Installing…");
    expect(installing.length).toBeGreaterThan(0);

    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({ action: "primary" })
    );
    // update_now keeps the window open — no dismiss bridge on this path.
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

    // Wait for the MSW response to replace the TEMP fallback logo urls.
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
