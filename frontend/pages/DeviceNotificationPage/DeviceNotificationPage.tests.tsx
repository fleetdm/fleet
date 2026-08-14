import React from "react";
import { screen, waitFor } from "@testing-library/react";

import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";
import {
  createMockNotificationView,
  customDeviceNotificationHandler,
  errorDeviceNotificationHandler,
  notFoundDeviceNotificationHandler,
} from "test/handlers/device-notifications-handlers";

import DeviceNotificationPage from "./DeviceNotificationPage";

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
});
