import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockDeviceSoftwareResponse } from "__mocks__/deviceUserMock";
import { createMockGetHostSoftwareResponse } from "__mocks__/hostMock";
import createMockUser from "__mocks__/userMock";
import { HostPlatform } from "interfaces/platform";
import deviceAPI from "services/entities/device_user";
import hostAPI from "services/entities/hosts";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import HostSoftware, { parseHostSoftwareQueryParams } from "./HostSoftware";

const PATHNAME = "/hosts/1/software";

const renderHostSoftware = ({
  platform,
  query = {},
  isMyDevicePage = false,
  isPremiumTier = false,
}: {
  platform: HostPlatform;
  query?: Parameters<typeof parseHostSoftwareQueryParams>[0];
  isMyDevicePage?: boolean;
  isPremiumTier?: boolean;
}) => {
  const replace = jest.fn();
  const router = createMockRouter({ replace });
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        currentUser: createMockUser(),
        // The My device page has no app context; it gets the tier as a prop.
        isPremiumTier: !isMyDevicePage && isPremiumTier,
      },
    },
  });
  const props = {
    id: isMyDevicePage ? "device-token" : 1,
    platform,
    router,
    pathname: PATHNAME,
    hostTeamId: 0,
    onShowInventoryVersions: noop,
    isSoftwareEnabled: true,
    isMyDevicePage,
    isPremiumTier: isMyDevicePage ? isPremiumTier : undefined,
  };
  const result = render(
    <HostSoftware
      {...props}
      queryParams={parseHostSoftwareQueryParams(query)}
    />
  );
  const rerender = (nextQuery: typeof query) =>
    result.rerender(
      <HostSoftware
        {...props}
        queryParams={parseHostSoftwareQueryParams(nextQuery)}
      />
    );
  return { replace, user: result.user, rerender };
};

const lastCallParams = (spy: jest.SpyInstance) =>
  spy.mock.calls[spy.mock.calls.length - 1][0];

describe("parseHostSoftwareQueryParams", () => {
  it("parses ai_tool", () => {
    expect(parseHostSoftwareQueryParams({ ai_tool: "true" }).ai_tool).toBe(
      true
    );
    expect(parseHostSoftwareQueryParams({}).ai_tool).toBeFalsy();
  });
});

describe("HostSoftware", () => {
  let getHostSoftware: jest.SpyInstance;

  beforeEach(() => {
    getHostSoftware = jest
      .spyOn(hostAPI, "getHostSoftware")
      .mockResolvedValue(createMockGetHostSoftwareResponse());
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("writes the macOS app default into the URL before requesting software", async () => {
    const { replace } = renderHostSoftware({
      platform: "darwin",
      query: { query: "chrome", vulnerable: "true" },
    });

    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.pathname).toBe(PATHNAME);
    expect(url.searchParams.get("types")).toBe("macos_app");
    expect(url.searchParams.get("query")).toBe("chrome");
    expect(url.searchParams.get("vulnerable")).toBe("true");
    // per_page is never read back from the URL.
    expect(url.searchParams.has("per_page")).toBe(false);
    expect(getHostSoftware).not.toHaveBeenCalled();
  });

  it("applies the macOS app default when the URL only has another platform's types", async () => {
    const { replace } = renderHostSoftware({
      platform: "darwin",
      query: { types: "windows_app", query: "chrome" },
    });

    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.get("types")).toBe("macos_app");
    expect(url.searchParams.get("query")).toBe("chrome");
    expect(getHostSoftware).not.toHaveBeenCalled();
  });

  it("requests top-level macOS apps when macOS app is selected", async () => {
    const { replace } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app" },
    });

    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
    const params = lastCallParams(getHostSoftware);
    expect(params).toEqual(
      expect.objectContaining({ source: "apps", macos_applications: true })
    );
    expect(params).not.toHaveProperty("types");
    expect(params).not.toHaveProperty("extension_for");
    expect(replace).not.toHaveBeenCalled();
  });

  it("drops macos_applications when Show helpers is on", async () => {
    renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app", macos_applications: "false" },
    });

    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
    expect(lastCallParams(getHostSoftware).macos_applications).toBeUndefined();
  });

  it("hides Show helpers and omits macos_applications without macOS app", async () => {
    renderHostSoftware({
      platform: "darwin",
      query: { types: "chrome_extension", macos_applications: "true" },
    });

    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
    const params = lastCallParams(getHostSoftware);
    expect(params).toEqual(
      expect.objectContaining({
        source: "chrome_extensions",
        extension_for: "chrome",
      })
    );
    expect(params.macos_applications).toBeUndefined();
    expect(
      screen.queryByRole("switch", { name: "Show helpers" })
    ).not.toBeInTheDocument();
  });

  it("re-applies the macOS app default when navigation strips the URL", async () => {
    // Clicking the Software tab pushes the bare tab path without remounting
    // the card, so an absent `types` must mean "default", not "cleared".
    const { replace, rerender } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app" },
    });
    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());

    rerender({});
    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.get("types")).toBe("macos_app");
    expect(getHostSoftware).toHaveBeenCalledTimes(1);
  });

  it("writes types=none when every type is deselected", async () => {
    const { replace, user } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app", macos_applications: "false" },
    });

    await user.click(await screen.findByRole("button", { name: "Filtered" }));
    await user.click(screen.getByRole("checkbox", { name: "macOS app" }));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.get("types")).toBe("none");
    expect(url.searchParams.has("macos_applications")).toBe(false);
  });

  it("treats types=none as a cleared selection, also after a reload", async () => {
    const { replace } = renderHostSoftware({
      platform: "darwin",
      query: { types: "none" },
    });

    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
    const params = lastCallParams(getHostSoftware);
    expect(params.source).toBeUndefined();
    expect(params.macos_applications).toBeUndefined();
    expect(
      await screen.findByRole("button", { name: "Add filters" })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("switch", { name: "Show helpers" })
    ).not.toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });

  it("keeps the URL when filters are applied unchanged", async () => {
    const { replace, user } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app", page: "2" },
    });

    await user.click(await screen.findByRole("button", { name: "Filtered" }));
    await user.click(screen.getByRole("button", { name: "Apply" }));

    expect(replace).not.toHaveBeenCalled();
  });

  it("resets Show helpers when macOS app is deselected", async () => {
    const { replace, user } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app", macos_applications: "false" },
    });

    await user.click(await screen.findByRole("button", { name: "Filtered" }));
    await user.click(screen.getByRole("checkbox", { name: "macOS app" }));
    await user.click(
      screen.getByRole("checkbox", { name: "Chrome extension" })
    );
    await user.click(screen.getByRole("button", { name: "Apply" }));

    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.get("types")).toBe("chrome_extension");
    expect(url.searchParams.has("macos_applications")).toBe(false);
  });

  it("neither redirects nor shows Show helpers on Windows", async () => {
    const { replace } = renderHostSoftware({ platform: "windows" });

    await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
    expect(lastCallParams(getHostSoftware).source).toBeUndefined();
    expect(
      screen.queryByRole("switch", { name: "Show helpers" })
    ).not.toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });

  it("offers only the host platform's inventory types", async () => {
    const { user } = renderHostSoftware({ platform: "windows" });

    await user.click(
      await screen.findByRole("button", { name: "Add filters" })
    );
    expect(
      screen.getByRole("checkbox", { name: "Windows app" })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "macOS app" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Script-only package (.ps1)" })
    ).not.toBeInTheDocument();
  });

  it("drops another platform's types from the URL before requesting software", async () => {
    const { replace } = renderHostSoftware({
      platform: "windows",
      query: { types: "macos_app", query: "chrome" },
    });

    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.has("types")).toBe(false);
    expect(url.searchParams.get("query")).toBe("chrome");
    expect(getHostSoftware).not.toHaveBeenCalled();
  });

  it("keeps only the host platform's types when the URL mixes platforms", async () => {
    const { replace } = renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app,windows_app" },
    });

    await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
    const url = new URL(replace.mock.calls[0][0], "http://fleet");
    expect(url.searchParams.get("types")).toBe("macos_app");
    expect(getHostSoftware).not.toHaveBeenCalled();
  });

  it("sends the selected types to the My device endpoint", async () => {
    const getDeviceSoftware = jest
      .spyOn(deviceAPI, "getDeviceSoftware")
      .mockResolvedValue(createMockDeviceSoftwareResponse());

    renderHostSoftware({
      platform: "darwin",
      query: { types: "macos_app" },
      isMyDevicePage: true,
    });

    await waitFor(() => expect(getDeviceSoftware).toHaveBeenCalled());
    const params = lastCallParams(getDeviceSoftware);
    expect(params).toEqual(
      expect.objectContaining({ source: "apps", macos_applications: true })
    );
    expect(params).not.toHaveProperty("types");
    expect(getHostSoftware).not.toHaveBeenCalled();
  });

  describe("AI tools filter", () => {
    it("narrows the macOS app default to AI tools", async () => {
      renderHostSoftware({
        platform: "darwin",
        query: { types: "macos_app", ai_tool: "true" },
        isPremiumTier: true,
      });

      await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
      expect(lastCallParams(getHostSoftware)).toEqual(
        expect.objectContaining({
          source: "apps",
          macos_applications: true,
          ai_tool: true,
        })
      );
    });

    it("requests every AI tool on the host when no type is selected", async () => {
      renderHostSoftware({
        platform: "darwin",
        query: { types: "none", ai_tool: "true" },
        isPremiumTier: true,
      });

      await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
      const params = lastCallParams(getHostSoftware);
      expect(params.ai_tool).toBe(true);
      expect(params.source).toBeUndefined();
    });

    it("works on a Windows host with no default type", async () => {
      const { replace } = renderHostSoftware({
        platform: "windows",
        query: { ai_tool: "true" },
        isPremiumTier: true,
      });

      await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
      const params = lastCallParams(getHostSoftware);
      expect(params.ai_tool).toBe(true);
      expect(params.source).toBeUndefined();
      expect(replace).not.toHaveBeenCalled();
    });

    it("omits ai_tool from the request while the toggle is off", async () => {
      renderHostSoftware({
        platform: "windows",
        isPremiumTier: true,
      });

      await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
      expect(lastCallParams(getHostSoftware).ai_tool).toBeUndefined();
    });

    it("writes ai_tool=true to the URL from the filters modal and resets the page", async () => {
      const { replace, user } = renderHostSoftware({
        platform: "darwin",
        query: { types: "macos_app", page: "2", query: "claude" },
        isPremiumTier: true,
      });

      await user.click(await screen.findByRole("button", { name: "Filtered" }));
      await user.click(screen.getByRole("switch", { name: "AI tools" }));
      await user.click(screen.getByRole("button", { name: "Apply" }));

      const url = new URL(replace.mock.calls[0][0], "http://fleet");
      expect(url.searchParams.get("ai_tool")).toBe("true");
      expect(url.searchParams.get("types")).toBe("macos_app");
      expect(url.searchParams.get("query")).toBe("claude");
      expect(url.searchParams.get("page")).toBe("0");
    });

    it("offers the AI types on a Premium host", async () => {
      const { user } = renderHostSoftware({
        platform: "linux",
        isPremiumTier: true,
      });

      await user.click(
        await screen.findByRole("button", { name: "Add filters" })
      );
      ["AI CLI tool", "AI skill", "MCP server"].forEach((name) =>
        expect(screen.getByRole("checkbox", { name })).toBeInTheDocument()
      );
    });

    it("hides the toggle and the AI types on Free", async () => {
      const { user } = renderHostSoftware({ platform: "windows" });

      await user.click(
        await screen.findByRole("button", { name: "Add filters" })
      );
      expect(
        screen.queryByRole("switch", { name: "AI tools" })
      ).not.toBeInTheDocument();
      ["AI CLI tool", "AI skill", "MCP server"].forEach((name) =>
        expect(screen.queryByRole("checkbox", { name })).not.toBeInTheDocument()
      );
    });

    it("omits ai_tool from the request on Free when the URL carries it", async () => {
      renderHostSoftware({
        platform: "windows",
        query: { types: "windows_app", ai_tool: "true" },
      });

      await waitFor(() => expect(getHostSoftware).toHaveBeenCalled());
      const params = lastCallParams(getHostSoftware);
      expect(params.ai_tool).toBeUndefined();
      expect(params.source).toBe("programs");
    });

    it("drops the AI types and ai_tool from the URL on Free", async () => {
      const { replace } = renderHostSoftware({
        platform: "windows",
        query: { types: "windows_app,ai_skill", ai_tool: "true" },
      });

      await waitFor(() => expect(replace).toHaveBeenCalledTimes(1));
      const url = new URL(replace.mock.calls[0][0], "http://fleet");
      expect(url.searchParams.get("types")).toBe("windows_app");
      expect(url.searchParams.has("ai_tool")).toBe(false);
      expect(getHostSoftware).not.toHaveBeenCalled();
    });

    it("sends ai_tool to the My device endpoint on Premium", async () => {
      const getDeviceSoftware = jest
        .spyOn(deviceAPI, "getDeviceSoftware")
        .mockResolvedValue(createMockDeviceSoftwareResponse());

      renderHostSoftware({
        platform: "darwin",
        query: { types: "macos_app", ai_tool: "true" },
        isMyDevicePage: true,
        isPremiumTier: true,
      });

      await waitFor(() => expect(getDeviceSoftware).toHaveBeenCalled());
      expect(lastCallParams(getDeviceSoftware)).toEqual(
        expect.objectContaining({ source: "apps", ai_tool: true })
      );
    });

    it("shows the toggle on My device on Premium", async () => {
      jest
        .spyOn(deviceAPI, "getDeviceSoftware")
        .mockResolvedValue(createMockDeviceSoftwareResponse());

      const { user } = renderHostSoftware({
        platform: "windows",
        isMyDevicePage: true,
        isPremiumTier: true,
      });

      await user.click(
        await screen.findByRole("button", { name: "Add filters" })
      );
      expect(
        screen.getByRole("switch", { name: "AI tools" })
      ).toBeInTheDocument();
    });

    it("hides the toggle and drops ai_tool on My device on Free", async () => {
      const getDeviceSoftware = jest
        .spyOn(deviceAPI, "getDeviceSoftware")
        .mockResolvedValue(createMockDeviceSoftwareResponse());

      const { user } = renderHostSoftware({
        platform: "windows",
        query: { ai_tool: "true" },
        isMyDevicePage: true,
      });

      await waitFor(() => expect(getDeviceSoftware).toHaveBeenCalled());
      expect(lastCallParams(getDeviceSoftware).ai_tool).toBeUndefined();

      await user.click(
        await screen.findByRole("button", { name: "Add filters" })
      );
      expect(
        screen.queryByRole("switch", { name: "AI tools" })
      ).not.toBeInTheDocument();
    });
  });
});
