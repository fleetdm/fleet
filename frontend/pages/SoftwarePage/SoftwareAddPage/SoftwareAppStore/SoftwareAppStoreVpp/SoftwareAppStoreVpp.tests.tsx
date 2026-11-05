import { screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import { createMockVppApp } from "__mocks__/appleMdm";
import { createMockVppToken } from "__mocks__/mdmMock";
import { IMdmVppToken } from "interfaces/mdm";
import PATHS from "router/paths";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { getPathWithQueryParams } from "utilities/url";

import SoftwareAppStoreVpp from "./SoftwareAppStoreVpp";

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

const router = createMockRouter();

const emptyVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({ vpp_tokens: [] });
});

const labelsHandler = http.get(baseUrl("/labels/summary"), () => {
  return HttpResponse.json({ labels: [] });
});

const teamMismatchVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({
    vpp_tokens: [
      {
        id: 1,
        org_name: "Test Org",
        location: "US",
        renew_date: "2027-01-01",
        teams: [{ team_id: 999, name: "Other fleet" }],
      },
    ],
  });
});

const teamVppHandler = http.get(baseUrl("/vpp_tokens"), () => {
  return HttpResponse.json({
    vpp_tokens: [
      {
        id: 1,
        org_name: "Test Org",
        location: "US",
        renew_date: "2027-01-01",
        teams: [{ team_id: 1, name: "Workstations" }],
      },
    ],
  });
});

describe("SoftwareAppStoreVpp", () => {
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("shows enable VPP when Apple MDM is off even if the fleet has a VPP token", async () => {
    let appsRequested = false;
    mockServer.use(
      teamVppHandler,
      labelsHandler,
      http.get(baseUrl("/software/app_store_apps"), () => {
        appsRequested = true;
        return HttpResponse.json({ app_store_apps: [] });
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          isMacMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });
    expect(
      screen.getByText("Enable VPP to add App Store apps (MDM required).")
    ).toBeInTheDocument();
    expect(appsRequested).toBe(false);
  });

  it("shows enable VPP button for admins when no VPP tokens exist", async () => {
    mockServer.use(emptyVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("Enable VPP to add App Store apps (MDM required).")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Enable VPP" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when no VPP tokens exist", async () => {
    mockServer.use(emptyVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: false,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Volume Purchasing Program \(VPP\) isn't enabled/i)
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("To add App Store apps, ask your admin to enable VPP.")
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable VPP" })
    ).not.toBeInTheDocument();
  });

  it("shows add fleet to VPP button for admins when team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={1}
        currentTeamName="Workstations"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Workstations fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, first add Workstations fleet to VPP."
      )
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Edit VPP" })
    ).toBeInTheDocument();
  });

  it("shows ask your admin copy for non-admins when team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: false,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={1}
        currentTeamName="Workstations"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Workstations fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, ask your admin to add Workstations fleet to VPP."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Edit VPP" })
    ).not.toBeInTheDocument();
  });

  it("shows unassigned-hosts copy when No team has no VPP token", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(
      <SoftwareAppStoreVpp
        currentTeamId={0}
        currentTeamName="No team"
        router={router}
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(
          /Unassigned hosts aren't in Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText(
        "To add App Store apps, first add unassigned hosts to VPP."
      )
    ).toBeInTheDocument();
  });

  it("falls back to generic copy when the fleet name hasn't resolved yet", async () => {
    mockServer.use(teamMismatchVppHandler, labelsHandler);

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isMacMdmEnabledAndConfigured: true,
          isGlobalAdmin: true,
        },
      },
    });

    render(<SoftwareAppStoreVpp currentTeamId={1} router={router} />);

    await waitFor(() => {
      expect(
        screen.getByText(
          /This fleet isn't added to Volume Purchasing Program \(VPP\)/i
        )
      ).toBeInTheDocument();
    });

    expect(
      screen.getByText("To add App Store apps, first add this fleet to VPP.")
    ).toBeInTheDocument();
  });

  describe("VPP token dropdown", () => {
    const US_LABEL = "Acme Inc. • 🇺🇸 United States";
    const JP_LABEL = "Acme K.K. • 🇯🇵 Japan";

    const usToken = createMockVppToken({
      id: 1,
      org_name: "Acme Inc.",
      location: "Acme Inc.",
      country_code: "us",
      teams: [{ team_id: 1, name: "Workstations" }],
    });
    const jpToken = createMockVppToken({
      id: 2,
      org_name: "Acme K.K.",
      location: "Acme K.K.",
      country_code: "jp",
      teams: [{ team_id: 1, name: "Workstations" }],
    });

    const tokensHandler = (tokens: IMdmVppToken[]) =>
      http.get(baseUrl("/vpp_tokens"), () =>
        HttpResponse.json({ vpp_tokens: tokens })
      );

    /** Pass a `client` to keep switched-away queries cached, as in the app. */
    const renderPage = (
      gitOpsModeEnabled = false,
      client?: QueryClient,
      teamId = 1,
      { isGlobalAdmin = true, pageRouter = router } = {}
    ) => {
      const render = createCustomRenderer({
        withBackendMock: !client,
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin,
            isMacMdmEnabledAndConfigured: true,
            config: { gitops: { gitops_mode_enabled: gitOpsModeEnabled } },
          },
        },
      });
      const page = (
        <SoftwareAppStoreVpp currentTeamId={teamId} router={pageRouter} />
      );
      return render(
        client ? (
          <QueryClientProvider client={client}>{page}</QueryClientProvider>
        ) : (
          page
        )
      );
    };

    /** Registers the apps endpoint and records every request URL. */
    const useAppsHandler = (appsByToken: Record<string, unknown> = {}) => {
      const requests: URL[] = [];
      mockServer.use(
        labelsHandler,
        http.get(baseUrl("/software/app_store_apps"), ({ request }) => {
          const url = new URL(request.url);
          requests.push(url);
          const tokenId = url.searchParams.get("vpp_token_id") ?? "";
          return HttpResponse.json({
            app_store_apps: appsByToken[tokenId] ?? [
              createMockVppApp({ name: `App ${tokenId}`, platform: "ios" }),
            ],
          });
        })
      );
      return requests;
    };

    const openTokenDropdown = async (
      user: ReturnType<typeof renderPage>["user"]
    ) => {
      await user.click(
        screen.getByRole("combobox", { name: "Select VPP token" })
      );
    };

    it("lists the fleet's tokens with org name subtext and selects the first added", async () => {
      mockServer.use(tokensHandler([jpToken, usToken]));
      const requests = useAppsHandler();

      const { user } = renderPage();

      expect(await screen.findByText(US_LABEL)).toBeInTheDocument();
      expect(screen.getByText("VPP token")).toBeInTheDocument();
      await waitFor(() => expect(requests).toHaveLength(1));
      expect(requests[0].searchParams.get("vpp_token_id")).toBe("1");
      expect(requests[0].searchParams.get("fleet_id")).toBe("1");

      await screen.findByRole("radio", { name: /App 1/ });
      await openTokenDropdown(user);
      const [first, second] = screen.getAllByTestId("dropdown-option");
      expect(within(first).getByText(US_LABEL)).toBeInTheDocument();
      expect(within(first).getByText("Acme Inc.")).toBeInTheDocument();
      expect(within(second).getByText(JP_LABEL)).toBeInTheDocument();
      expect(within(second).getByText("Acme K.K.")).toBeInTheDocument();
    });

    it("refetches the app list with the picked token and clears the selected app", async () => {
      mockServer.use(tokensHandler([usToken, jpToken]));
      const canva = createMockVppApp({
        app_store_id: "10",
        name: "Canva",
        platform: "ios",
      });
      const requests = useAppsHandler({ "1": [canva], "2": [canva] });

      const { user } = renderPage(
        false,
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      );

      await user.click(await screen.findByRole("radio", { name: /Canva/ }));
      expect(screen.getByRole("radio", { name: /Canva/ })).toBeChecked();

      await openTokenDropdown(user);
      await user.click(screen.getByText(JP_LABEL));

      await waitFor(() => expect(requests).toHaveLength(2));
      expect(requests[1].searchParams.get("vpp_token_id")).toBe("2");
      await user.click(await screen.findByRole("radio", { name: /Canva/ }));

      // The first token's list is cached, so only the form's key resets the selection.
      await openTokenDropdown(user);
      await user.click(screen.getByText(US_LABEL));
      expect(
        await screen.findByRole("radio", { name: /Canva/ })
      ).not.toBeChecked();
    });

    it("hides the dropdown with one token and still sends its id", async () => {
      mockServer.use(tokensHandler([usToken]));
      const requests = useAppsHandler();

      renderPage();

      await waitFor(() => expect(requests).toHaveLength(1));
      expect(requests[0].searchParams.get("vpp_token_id")).toBe("1");
      expect(
        screen.queryByRole("combobox", { name: "Select VPP token" })
      ).not.toBeInTheDocument();
    });

    it("offers an All fleets token alongside a fleet token", async () => {
      mockServer.use(
        tokensHandler([
          createMockVppToken({ ...usToken, teams: [] }),
          jpToken,
          createMockVppToken({
            id: 3,
            teams: [{ team_id: 999, name: "Other fleet" }],
          }),
        ])
      );
      useAppsHandler();

      const { user } = renderPage();

      await screen.findByRole("radio", { name: /App 1/ });
      await openTokenDropdown(user);
      expect(screen.getAllByTestId("dropdown-option")).toHaveLength(2);
    });

    const EXPIRED = "2020-01-01T00:00:00Z";
    const expiredUs = createMockVppToken({ ...usToken, renew_date: EXPIRED });
    const expiredJp = createMockVppToken({ ...jpToken, renew_date: EXPIRED });

    it("lists an expired token, and switching between it and a valid one shows the empty state then the apps", async () => {
      mockServer.use(tokensHandler([expiredUs, jpToken]));
      const requests = useAppsHandler();

      const { user } = renderPage(
        false,
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      );

      expect(await screen.findByText("Token expired")).toBeInTheDocument();
      expect(requests).toHaveLength(0);

      await openTokenDropdown(user);
      expect(screen.getAllByTestId("dropdown-option")).toHaveLength(2);
      await user.click(screen.getByText(JP_LABEL));
      await screen.findByRole("radio", { name: /App 2/ });
      expect(screen.queryByText("Token expired")).not.toBeInTheDocument();
      expect(requests).toHaveLength(1);
      expect(requests[0].searchParams.get("vpp_token_id")).toBe("2");

      await openTokenDropdown(user);
      await user.click(screen.getByText(US_LABEL));
      expect(await screen.findByText("Token expired")).toBeInTheDocument();
      expect(requests).toHaveLength(1);
    });

    it("selects an expired token by default when every token has expired and keeps the dropdown", async () => {
      mockServer.use(tokensHandler([expiredUs, expiredJp]));
      const requests = useAppsHandler();

      renderPage();

      expect(await screen.findByText("Token expired")).toBeInTheDocument();
      expect(
        screen.getByRole("combobox", { name: "Select VPP token" })
      ).toBeInTheDocument();
      expect(requests).toHaveLength(0);
    });

    it("hides the dropdown and shows the empty state with one expired token", async () => {
      mockServer.use(tokensHandler([expiredUs]));
      const requests = useAppsHandler();

      renderPage();

      expect(await screen.findByText("Token expired")).toBeInTheDocument();
      expect(
        screen.queryByRole("combobox", { name: "Select VPP token" })
      ).not.toBeInTheDocument();
      expect(requests).toHaveLength(0);
    });

    it("lets a global admin renew an expired token from the VPP settings page", async () => {
      mockServer.use(tokensHandler([expiredUs]));
      useAppsHandler();
      const push = jest.fn();

      const { user } = renderPage(false, undefined, 1, {
        pageRouter: createMockRouter({ push }),
      });

      expect(
        await screen.findByText(
          "To add apps from this token you must renew it first."
        )
      ).toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: "Renew" }));
      expect(push).toHaveBeenCalledWith(PATHS.ADMIN_INTEGRATIONS_VPP);
    });

    it("keeps a disabled Add software and a working Cancel for an expired token", async () => {
      mockServer.use(tokensHandler([expiredUs]));
      useAppsHandler();
      const push = jest.fn();

      const { user } = renderPage(false, undefined, 1, {
        pageRouter: createMockRouter({ push }),
      });

      expect(await screen.findByText("Token expired")).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Add software" })
      ).toBeDisabled();
      await user.click(screen.getByRole("button", { name: "Cancel" }));
      expect(push).toHaveBeenCalledWith(
        getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, { fleet_id: 1 })
      );
    });

    it("tells a non-admin to ask their admin to renew an expired token", async () => {
      mockServer.use(tokensHandler([expiredUs]));
      useAppsHandler();

      renderPage(false, undefined, 1, { isGlobalAdmin: false });

      expect(
        await screen.findByText(
          "To add apps from this token, ask your admin to renew it."
        )
      ).toBeInTheDocument();
      expect(screen.getByText("Token expired")).toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "Renew" })
      ).not.toBeInTheDocument();
    });

    it("keeps the dropdown when the picked token has no apps", async () => {
      mockServer.use(tokensHandler([usToken, jpToken]));
      mockServer.use(
        labelsHandler,
        http.get(baseUrl("/software/app_store_apps"), () =>
          HttpResponse.json({ app_store_apps: null })
        )
      );

      renderPage();

      expect(
        await screen.findByText("You don't have any App Store apps")
      ).toBeInTheDocument();
      expect(
        screen.getByRole("combobox", { name: "Select VPP token" })
      ).toBeInTheDocument();
    });

    it("requests Unassigned apps with fleet_id 0 and the token's id", async () => {
      mockServer.use(
        tokensHandler([
          createMockVppToken({
            ...usToken,
            teams: [{ team_id: 0, name: "Unassigned" }],
          }),
        ])
      );
      const requests = useAppsHandler();

      renderPage(false, undefined, 0);

      await waitFor(() => expect(requests).toHaveLength(1));
      expect(requests[0].searchParams.get("fleet_id")).toBe("0");
      expect(requests[0].searchParams.get("vpp_token_id")).toBe("1");
    });

    it("adds the app with the selected token's id", async () => {
      mockServer.use(tokensHandler([usToken, jpToken]));
      useAppsHandler();
      let body: Record<string, unknown> | undefined;
      mockServer.use(
        http.post(baseUrl("/software/app_store_apps"), async ({ request }) => {
          body = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json({ software_title_id: 5 });
        })
      );

      const { user } = renderPage();

      await screen.findByRole("radio", { name: /App 1/ });
      await openTokenDropdown(user);
      await user.click(screen.getByText(JP_LABEL));
      await user.click(await screen.findByRole("radio", { name: /App 2/ }));
      await user.click(screen.getByRole("button", { name: "Add software" }));

      await waitFor(() => expect(body).toBeDefined());
      expect(body).toMatchObject({ fleet_id: 1, vpp_token_id: 2 });
    });

    it("adds the app with the only token's id when the dropdown is hidden", async () => {
      mockServer.use(tokensHandler([jpToken]));
      useAppsHandler();
      let body: Record<string, unknown> | undefined;
      mockServer.use(
        http.post(baseUrl("/software/app_store_apps"), async ({ request }) => {
          body = (await request.json()) as Record<string, unknown>;
          return HttpResponse.json({ software_title_id: 5 });
        })
      );

      const { user } = renderPage();

      await user.click(await screen.findByRole("radio", { name: /App 2/ }));
      await user.click(screen.getByRole("button", { name: "Add software" }));

      await waitFor(() => expect(body).toBeDefined());
      expect(body).toMatchObject({ vpp_token_id: 2 });
    });

    it("still switches lists in GitOps mode while Add software stays disabled", async () => {
      mockServer.use(tokensHandler([usToken, jpToken]));
      const requests = useAppsHandler();

      const { user } = renderPage(true);

      await screen.findByRole("radio", { name: /App 1/ });
      await openTokenDropdown(user);
      await user.click(screen.getByText(JP_LABEL));

      await waitFor(() => expect(requests).toHaveLength(2));
      expect(requests[1].searchParams.get("vpp_token_id")).toBe("2");
      await user.click(await screen.findByRole("radio", { name: /App 2/ }));
      expect(
        screen.getByRole("button", { name: "Add software" })
      ).toBeDisabled();
    });
  });
});
