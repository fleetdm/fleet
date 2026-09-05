import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";

import OrgSettingsPage from "./OrgSettingsPage";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const configUrl = baseUrl("/config");

const createGetConfigHandler = (
  getSpy: jest.Mock,
  overrides?: Partial<IConfig>
) => {
  return http.get(configUrl, () => {
    getSpy();
    return HttpResponse.json(createMockConfig(overrides));
  });
};

const createPatchConfigHandler = (
  patchSpy: jest.Mock,
  overrides?: Partial<IConfig>
) => {
  return http.patch(configUrl, async ({ request }) => {
    const body = (await request.json()) as Partial<IConfig>;
    patchSpy(body);
    return HttpResponse.json(
      createMockConfig({
        ...overrides,
        org_info: {
          ...createMockConfig(overrides).org_info,
          ...body.org_info,
        },
      })
    );
  });
};

describe("OrgSettingsPage", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isPremiumTier: true, setConfig: jest.fn() },
    },
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("does not refetch /config after a successful save", async () => {
    // The read-after-write regression (#42546): on save, the page used to
    // refetch /config, which returns stale data under DB read-replica lag.
    // The fix consumes the PATCH response directly. This asserts GET /config
    // fires ONCE — for the initial load — and never again after the write.
    const getSpy = jest.fn();
    const patchSpy = jest.fn();
    mockServer.use(
      createGetConfigHandler(getSpy, {
        org_info: {
          org_name: "Old name",
          org_logo_url: "",
          org_logo_url_light_background: "",
          contact_url: "https://fleetdm.com/company/contact",
        },
      }),
      createPatchConfigHandler(patchSpy)
    );

    const { user } = render(
      <OrgSettingsPage
        params={{ section: "organization" }}
        router={createMockRouter()}
      />
    );

    const orgNameInput = await screen.findByLabelText("Organization name");
    await user.clear(orgNameInput);
    await user.type(orgNameInput, "New name");

    const saveButton = screen.getByRole("button", { name: /save/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(patchSpy).toHaveBeenCalledTimes(1);
    });
    expect(patchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        org_info: expect.objectContaining({ org_name: "New name" }),
      })
    );

    // The critical assertion: GET /config fires only on initial mount, not
    // after the write. Give React Query a beat to schedule any (unwanted)
    // background refetch before asserting.
    await new Promise((resolve) => {
      setTimeout(resolve, 50);
    });
    expect(getSpy).toHaveBeenCalledTimes(1);
  });
});
