import { http, HttpResponse } from "msw";

import { baseUrl } from "test/test-utils";
import {
  INotificationAction,
  INotificationItem,
  INotificationView,
} from "interfaces/device_notification";

const notificationUrl = baseUrl("/device/:token/notifications/:uuid");
const notificationActionsUrl = baseUrl(
  "/device/:token/notifications/:uuid/actions"
);

const HOUR_ACTIONS: INotificationAction[] = [
  { id: "remind", label: "Remind me in 1 hour" },
  { id: "update_now", label: "Update now" },
];

const INSTALLING_ACTIONS: INotificationAction[] = [
  { id: "dismiss", label: "Hide" },
];

const createMockNotificationItem = (
  overrides?: Partial<INotificationItem>
): INotificationItem => ({
  software_title_id: 1,
  name: "1Password 8",
  display_name: "1Password",
  icon_url: null,
  ...overrides,
});

export const createMockNotificationView = (
  overrides?: Partial<INotificationView>
): INotificationView => ({
  uuid: "notif-uuid-1",
  org_logo_url_light_mode: "https://example.com/fleet-logo-light.png",
  org_logo_url_dark_mode: "https://example.com/fleet-logo-dark.png",
  title: "These apps will close and update in **1 hour**",
  description: "Save your work 💾",
  items: [
    createMockNotificationItem({
      software_title_id: 1,
      name: "1Password 8",
      display_name: "1Password",
    }),
    createMockNotificationItem({
      software_title_id: 2,
      name: "Slack",
      display_name: "Slack",
    }),
    createMockNotificationItem({
      software_title_id: 3,
      name: "Docker Desktop",
      display_name: "Docker Desktop",
    }),
  ],
  actions: HOUR_ACTIONS,
  ...overrides,
});

/** 20 items — exercises the four-and-a-half-row scroll case. */
export const createMockScrollingNotificationView = (): INotificationView =>
  createMockNotificationView({
    items: Array.from({ length: 20 }, (_, i) =>
      createMockNotificationItem({
        software_title_id: i + 1,
        name: `App ${i + 1}`,
        display_name: `App ${i + 1}`,
      })
    ),
  });

/** Post-update_now state: items show "Installing…", one Hide action. */
export const createMockInstallingNotificationView = (): INotificationView =>
  createMockNotificationView({
    items: createMockNotificationView().items.map((item) => ({
      ...item,
      status: "Installing…",
    })),
    actions: INSTALLING_ACTIONS,
  });

export const defaultDeviceNotificationHandler = http.get(notificationUrl, () =>
  HttpResponse.json(createMockNotificationView())
);

export const customDeviceNotificationHandler = (
  overrides?: Partial<INotificationView>
) =>
  http.get(notificationUrl, () =>
    HttpResponse.json(createMockNotificationView(overrides))
  );

export const scrollingDeviceNotificationHandler = http.get(
  notificationUrl,
  () => HttpResponse.json(createMockScrollingNotificationView())
);

export const installingDeviceNotificationHandler = http.get(
  notificationUrl,
  () => HttpResponse.json(createMockInstallingNotificationView())
);

export const notFoundDeviceNotificationHandler = http.get(notificationUrl, () =>
  HttpResponse.json(
    { errors: [{ name: "base", reason: "Not found" }] },
    { status: 404 }
  )
);

export const errorDeviceNotificationHandler = http.get(notificationUrl, () =>
  HttpResponse.json(
    { errors: [{ name: "base", reason: "Internal Server Error" }] },
    { status: 500 }
  )
);

/** POST /actions — echoes the update_now → Installing… transition. */
export const defaultDeviceNotificationActionHandler = http.post(
  notificationActionsUrl,
  async ({ request }) => {
    const body = (await request.json()) as { action: string };
    if (body.action === "update_now") {
      return HttpResponse.json(createMockInstallingNotificationView());
    }
    return HttpResponse.json(createMockNotificationView());
  }
);

export const errorDeviceNotificationActionHandler = http.post(
  notificationActionsUrl,
  () =>
    HttpResponse.json(
      { errors: [{ name: "base", reason: "Internal Server Error" }] },
      { status: 500 }
    )
);
