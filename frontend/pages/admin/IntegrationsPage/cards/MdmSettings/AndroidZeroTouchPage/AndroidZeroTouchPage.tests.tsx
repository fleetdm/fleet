import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { QueryClient, QueryClientProvider } from "react-query";

import createMockAxiosError from "__mocks__/axiosError";
import createMockUser from "__mocks__/userMock";
import mdmAndroidAPI from "services/entities/mdm_android";
import { createCustomRenderer } from "test/test-utils";

import AndroidZeroTouchPage from "./AndroidZeroTouchPage";

const CONFIGURED_APP_CONTEXT = {
  currentUser: createMockUser(),
  availableTeams: [
    { id: -1, name: "All fleets" },
    { id: 1, name: "Workstations" },
    { id: 0, name: "Unassigned" },
  ],
  isPremiumTier: true,
  isAndroidMdmEnabledAndConfigured: true,
};

const getNameField = () => screen.getByRole("textbox", { name: "Name" });
const getFleetPicker = () =>
  screen.getByRole("combobox", {
    name: "Pick the fleet that Android hosts will automatically enroll into",
  });
const getFleetOptions = () => screen.getAllByTestId("dropdown-option");
const selectFleet = async (name: string) => {
  await userEvent.click(getFleetPicker());
  const option = getFleetOptions().find((el) => el.textContent === name);
  if (!option) {
    throw new Error(`No fleet option named "${name}"`);
  }
  await userEvent.click(option);
};

describe("AndroidZeroTouchPage", () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  // `isPremiumTier` is undefined until the config request resolves. Neither the
  // paywall nor the premium-only request may appear in that window.
  test("shows neither the paywall nor a request until the tier is known", async () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { currentUser: createMockUser(), isPremiumTier: undefined },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.queryByText(/Fleet Premium/i)).toBeNull();
    expect(getConfig).not.toHaveBeenCalled();
    expect(await screen.findByTestId("spinner")).toBeVisible();
  });

  test("shows premium upsell without calling the API when not premium tier", () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...CONFIGURED_APP_CONTEXT,
          isPremiumTier: false,
        },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Fleet Premium/i)).toBeVisible();
    expect(getConfig).not.toHaveBeenCalled();
  });

  test("shows the prerequisite message when Android MDM is off", () => {
    const getConfig = jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration");

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          ...CONFIGURED_APP_CONTEXT,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(screen.getByText(/first turn on Android MDM/i)).toBeVisible();
    expect(screen.queryByText("DPC extras")).toBeNull();
    expect(getConfig).not.toHaveBeenCalled();
  });

  test("shows DPC extras after loading", async () => {
    jest.spyOn(mdmAndroidAPI, "getZeroTouchConfiguration").mockResolvedValue({
      test: "dpc-extras-json",
    });

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    await waitFor(() => {
      expect(screen.getByText(/dpc-extras-json/)).toBeVisible();
    });

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText(/Android zero-touch portal/)).toBeVisible();
    expect(getFleetPicker()).toBeEnabled();
    expect(getNameField()).toHaveValue("Unassigned");
    expect(screen.getByText(/use this name and pick/)).toBeVisible();
    expect(screen.queryByText(/coming soon/i)).toBeNull();
    expect(screen.getByRole("button", { name: "Copy name" })).toBeEnabled();
    expect(
      screen.getByRole("button", { name: "Copy DPC extras" })
    ).toBeEnabled();
    expect(mdmAndroidAPI.getZeroTouchConfiguration).toHaveBeenCalledWith(0);
  });

  test("shows error state on API failure and disables the DPC extras copy button", async () => {
    jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockRejectedValue(createMockAxiosError({ status: 403 }));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    await waitFor(() => {
      expect(screen.getByText(/gone wrong/i)).toBeVisible();
    });

    expect(screen.getByText("Android zero-touch")).toBeVisible();
    expect(screen.getByText("DPC extras")).toBeVisible();
    expect(
      screen.getByRole("button", { name: "Copy DPC extras" })
    ).toBeDisabled();
    expect(getNameField()).toHaveValue("Unassigned");
    expect(getFleetPicker()).toBeEnabled();
  });

  test("switching fleets requests that fleet's DPC extras and updates the name", async () => {
    const getConfig = jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockImplementation((fleetId) =>
        Promise.resolve({ token: `token-for-fleet-${fleetId}` })
      );

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    await screen.findByText(/token-for-fleet-0/);

    await userEvent.click(getFleetPicker());
    expect(getFleetOptions().map((el) => el.textContent)).toEqual([
      "Unassigned",
      "Workstations",
    ]);
    await userEvent.keyboard("{Escape}");

    await selectFleet("Workstations");

    await screen.findByText(/token-for-fleet-1/);
    expect(getConfig).toHaveBeenLastCalledWith(1);
    expect(getNameField()).toHaveValue("Workstations");
    expect(screen.queryByText(/token-for-fleet-0/)).toBeNull();
  });

  test("shows the spinner while refetching a previously viewed fleet", async () => {
    let resolveRefetch: (value: Record<string, unknown>) => void = () =>
      undefined;
    const getConfig = jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockImplementationOnce(() => Promise.resolve({ token: "unassigned-1" }))
      .mockImplementationOnce(() => Promise.resolve({ token: "workstations" }))
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveRefetch = resolve;
          })
      );

    // The shared renderer's client uses cacheTime: 0, which drops the
    // previously viewed fleet's data and hides the case under test.
    const cachingClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const render = createCustomRenderer({
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(
      <QueryClientProvider client={cachingClient}>
        <AndroidZeroTouchPage />
      </QueryClientProvider>
    );

    await screen.findByText(/unassigned-1/);
    await selectFleet("Workstations");
    await screen.findByText(/workstations/);

    await selectFleet("Unassigned");

    expect(await screen.findByTestId("spinner")).toBeVisible();
    expect(screen.queryByText(/unassigned-1/)).toBeNull();
    expect(getFleetPicker()).toBeDisabled();
    expect(getConfig).toHaveBeenCalledTimes(3);

    resolveRefetch({ token: "unassigned-2" });
    await screen.findByText(/unassigned-2/);
    expect(getFleetPicker()).toBeEnabled();
  });

  test("shows the error without retrying when the server fails", async () => {
    const getConfig = jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockRejectedValue(createMockAxiosError({ status: 500 }));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(await screen.findByText(/gone wrong/i)).toBeVisible();
    expect(getConfig).toHaveBeenCalledTimes(1);
    expect(getFleetPicker()).toBeEnabled();
  });

  test("disables the fleet picker and copy buttons while the request is pending", async () => {
    jest
      .spyOn(mdmAndroidAPI, "getZeroTouchConfiguration")
      .mockReturnValue(new Promise(() => undefined));

    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: CONFIGURED_APP_CONTEXT,
      },
    });

    render(<AndroidZeroTouchPage />);

    expect(await screen.findByTestId("spinner")).toBeVisible();
    expect(getFleetPicker()).toBeDisabled();
    expect(screen.getByRole("button", { name: "Copy name" })).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Copy DPC extras" })
    ).toBeDisabled();
  });
});
