import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

const router = createMockRouter();

describe("SoftwareAppStoreAndroid", () => {
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("shows enable button for admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText("To add Android apps, first enable Android MDM.")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Enable Android MDM" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: false,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText(
        "To add Android apps, ask your admin to enable Android MDM."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable Android MDM" })
    ).not.toBeInTheDocument();
  });

  it("shows the Android form with the fleet's custom labels as targets when Android MDM is enabled", async () => {
    let requestedFleetId: string | null = null;
    mockServer.use(
      http.get(baseUrl("/labels/summary"), ({ request }) => {
        requestedFleetId = new URL(request.url).searchParams.get("fleet_id");
        return HttpResponse.json({
          labels: [
            { id: 1, name: "All Hosts", label_type: "builtin" },
            { id: 2, name: "Android pilot", label_type: "regular" },
          ],
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={1} router={router} />
    );

    await user.click(await screen.findByRole("radio", { name: "Custom" }));

    expect(
      screen.getByRole("checkbox", { name: "Android pilot" })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "All Hosts" })
    ).not.toBeInTheDocument();
    expect(requestedFleetId).toBe("1");
    expect(
      screen.queryByText("Android MDM isn't enabled")
    ).not.toBeInTheDocument();
  });

  it("requests global labels for the Unassigned fleet", async () => {
    let requestedFleetId: string | null = null;
    mockServer.use(
      http.get(baseUrl("/labels/summary"), ({ request }) => {
        requestedFleetId = new URL(request.url).searchParams.get("fleet_id");
        return HttpResponse.json({
          labels: [{ id: 2, name: "Android pilot", label_type: "regular" }],
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={0} router={router} />
    );

    await user.click(await screen.findByRole("radio", { name: "Custom" }));

    expect(
      screen.getByRole("checkbox", { name: "Android pilot" })
    ).toBeInTheDocument();
    expect(requestedFleetId).toBe("global");
  });

  it("shows the form while custom labels load, with a spinner in place of the label list", async () => {
    let releaseLabels: () => void = () => undefined;
    const labelsReleased = new Promise<void>((resolve) => {
      releaseLabels = resolve;
    });
    mockServer.use(
      http.get(baseUrl("/labels/summary"), async () => {
        await labelsReleased;
        return HttpResponse.json({
          labels: [{ id: 2, name: "Android pilot", label_type: "regular" }],
        });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={1} router={router} />
    );

    expect(
      screen.getByRole("textbox", { name: "Application ID" })
    ).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(await screen.findByTestId("spinner")).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Android pilot" })
    ).not.toBeInTheDocument();

    releaseLabels();

    expect(
      await screen.findByRole("checkbox", { name: "Android pilot" })
    ).toBeInTheDocument();
    expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
  });

  it("keeps All hosts usable when custom labels fail to load", async () => {
    let addAppBody: Record<string, unknown> | undefined;
    mockServer.use(
      http.get(baseUrl("/labels/summary"), () =>
        HttpResponse.json({}, { status: 404 })
      ),
      http.post(baseUrl("/software/app_store_apps"), async ({ request }) => {
        addAppBody = (await request.json()) as Record<string, unknown>;
        return HttpResponse.json({ software_title_id: 7, name: "Gmail" });
      })
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    const { user } = render(
      <SoftwareAppStoreAndroid currentTeamId={1} router={router} />
    );

    // Custom shows the labels error in place of the label list
    await user.click(screen.getByRole("radio", { name: "Custom" }));
    expect(
      await screen.findByText(/Something's gone wrong/)
    ).toBeInTheDocument();

    // All hosts still submits, without labels
    await user.click(screen.getByRole("radio", { name: "All hosts" }));
    await user.type(
      screen.getByRole("textbox", { name: "Application ID" }),
      "com.google.android.gm"
    );
    await user.click(screen.getByRole("button", { name: "Add software" }));

    await waitFor(() => expect(addAppBody).toBeDefined());
    expect(addAppBody).toMatchObject({
      app_store_id: "com.google.android.gm",
      platform: "android",
    });
    expect(addAppBody).not.toHaveProperty("labels_include_any");
  });
});
