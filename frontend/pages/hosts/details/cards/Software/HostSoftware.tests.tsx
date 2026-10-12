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
}: {
  platform: HostPlatform;
  query?: Parameters<typeof parseHostSoftwareQueryParams>[0];
  isMyDevicePage?: boolean;
}) => {
  const replace = jest.fn();
  const router = createMockRouter({ replace });
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isGlobalAdmin: true, currentUser: createMockUser() },
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
});
