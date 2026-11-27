import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import {
  createMockAppStoreApp,
  createMockAppStoreAppAndroid,
  createMockAppStoreAppVersion,
  createMockSoftwarePackage,
  createMockSoftwarePackageIos,
  createMockSoftwareTitle,
  createMockSoftwareTitleDetails,
} from "__mocks__/softwareMock";
import { InstallerType } from "interfaces/software";
import { emptySelfServiceCategoriesHandler } from "test/handlers/self-service-categories-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer } from "test/test-utils";

import DeleteSoftwareModal from "./DeleteSoftwareModal/DeleteSoftwareModal";
import EditAutoUpdateConfigModal from "./EditAutoUpdateConfigModal/EditAutoUpdateConfigModal";
import EditConfigurationModal from "./EditConfigurationModal/EditConfigurationModal";
import EditIconModal from "./EditIconModal/EditIconModal";
import EditSoftwareModal from "./EditSoftwareModal/EditSoftwareModal";
import EditVersionModal from "./EditVersionModal/EditVersionModal";
import VersionsModal from "./VersionsModal/VersionsModal";

// Fire the debounced validator synchronously so tests don't wait for it.
jest.mock("use-debounce", () => ({
  useDebouncedCallback: <T extends (...args: unknown[]) => unknown>(fn: T) => {
    const callback = (...args: Parameters<T>) => fn(...args);
    callback.cancel = () => undefined;
    callback.flush = () => undefined;
    return callback;
  },
}));

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

interface IRecordedRequest {
  method: string;
  pathname: string;
  search: string;
  body: string;
}

const BASE = "/api/latest/fleet/software/titles";
let requests: IRecordedRequest[] = [];

const record = async ({ request }: { request: Request }) => {
  const url = new URL(request.url);
  requests.push({
    method: request.method,
    pathname: url.pathname,
    search: url.search,
    body: await request.text(),
  });
  return new HttpResponse(null, { status: 204 });
};

const expectOnlyRequest = (method: string, pathname: string) => {
  expect(requests).toHaveLength(1);
  const [req] = requests;
  expect(req.method).toBe(method);
  expect(req.pathname).toBe(pathname);
  expect(req.search).not.toMatch(/installer_id|version_id/);
  expect(req.body).not.toMatch(/installer_id|version_id/);
};

const premiumRender = createCustomRenderer({
  withBackendMock: true,
  context: {
    app: {
      isPremiumTier: true,
      isGlobalAdmin: true,
      config: createMockConfig({
        gitops: {
          gitops_mode_enabled: false,
          repository_url: "",
          exceptions: { labels: false, software: false, secrets: false },
        },
      }),
    },
  },
});

describe("per-item software URLs", () => {
  beforeEach(() => {
    requests = [];
    mockServer.use(
      emptySelfServiceCategoriesHandler,
      http.patch(`${BASE}/*`, record),
      http.delete(`${BASE}/*`, record)
    );
  });

  it("EditSoftwareModal PATCHes a package by its package id", async () => {
    const { user } = premiumRender(
      <EditSoftwareModal
        softwareId={5}
        teamId={1}
        softwareInstaller={createMockSoftwarePackage({
          installer_id: 11,
          self_service: false,
        })}
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
        installerType="package"
        name="App"
        displayName="App"
      />
    );
    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/5/packages/11`);
  });

  it("EditSoftwareModal PATCHes an App Store app by its version id", async () => {
    const { user } = premiumRender(
      <EditSoftwareModal
        softwareId={5}
        teamId={1}
        softwareInstaller={createMockAppStoreApp({
          id: 12,
          self_service: false,
        })}
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
        installerType="app-store"
        name="App"
        displayName="App"
      />
    );
    await user.click(screen.getByRole("switch"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/5/app_store_apps/12`);
  });

  it("EditConfigurationModal PATCHes an in-house app by its package id", async () => {
    const { user } = premiumRender(
      <EditConfigurationModal
        softwareId={6}
        teamId={1}
        softwareInstaller={createMockSoftwarePackageIos({ installer_id: 21 })}
        isApplePlatform
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/6/packages/21`);
  });

  it("EditConfigurationModal PATCHes a Google Play app by its version id", async () => {
    const { user } = premiumRender(
      <EditConfigurationModal
        softwareId={6}
        teamId={1}
        softwareInstaller={createMockAppStoreAppAndroid({
          id: 22,
          configuration: undefined,
        })}
        isApplePlatform={false}
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/6/app_store_apps/22`);
  });

  it("VersionsModal PATCHes the FMA pin by its package id", async () => {
    const { user } = premiumRender(
      <VersionsModal
        softwareTitle={createMockSoftwareTitle({
          packages: [
            createMockSoftwarePackage({
              installer_id: 31,
              fleet_maintained_app_id: 5,
              version: "149.0.2",
              pinned_version: null,
              fleet_maintained_versions: [
                {
                  id: 1,
                  version: "149.0.2",
                  filename: "a.pkg",
                  uploaded_at: "2026-01-02T00:00:00Z",
                },
                {
                  id: 2,
                  version: "148.0.1",
                  filename: "b.pkg",
                  uploaded_at: "2026-01-01T00:00:00Z",
                },
              ],
            }),
          ],
        })}
        softwareId={7}
        teamId={1}
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
      />
    );
    await user.click(screen.getByRole("radio", { name: "Pin to 148.0.1" }));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/7/packages/31`);
  });

  it("EditAutoUpdateConfigModal PATCHes the first App Store version", async () => {
    const { user } = premiumRender(
      <EditAutoUpdateConfigModal
        softwareTitle={createMockSoftwareTitleDetails({
          id: 8,
          app_store_apps: [createMockAppStoreAppVersion({ id: 41 })],
        })}
        teamId={1}
        refetchSoftwareTitle={jest.fn()}
        onExit={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/8/app_store_apps/41`);
  });

  it("EditVersionModal PATCHes the edited version by its id", async () => {
    const { user } = premiumRender(
      <EditVersionModal
        softwareId={9}
        teamId={1}
        version={createMockAppStoreAppVersion({ id: 51, name: "Prod" })}
        titleDisplayName="App"
        siblingVersionNames={[]}
        onExit={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: /^Save$/i }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/9/app_store_apps/51`);
  });

  it("EditIconModal saves the display name through the first package's URL", async () => {
    const { user } = premiumRender(
      <EditIconModal
        softwareId={10}
        teamIdForApi={1}
        software={createMockSoftwarePackage({ installer_id: 61 })}
        onExit={jest.fn()}
        refetchSoftwareTitle={jest.fn()}
        iconUploadedAt=""
        setIconUploadedAt={jest.fn()}
        installerType={"package" as InstallerType}
        previewInfo={{
          type: "apps",
          source: "apps",
          currentIconUrl: null,
          name: "App",
          titleName: "App",
          versions: 1,
        }}
      />
    );
    await user.type(screen.getByLabelText("Display name"), "New name");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/10/packages/61`);
  });

  it("EditIconModal saves the display name through the first App Store version's URL", async () => {
    const { user } = premiumRender(
      <EditIconModal
        softwareId={10}
        teamIdForApi={1}
        software={createMockAppStoreApp({ id: 62 })}
        onExit={jest.fn()}
        refetchSoftwareTitle={jest.fn()}
        iconUploadedAt=""
        setIconUploadedAt={jest.fn()}
        installerType="app-store"
        previewInfo={{
          type: "apps",
          source: "apps",
          currentIconUrl: null,
          name: "App",
          titleName: "App",
          versions: 1,
        }}
      />
    );
    await user.type(screen.getByLabelText("Display name"), "New name");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("PATCH", `${BASE}/10/app_store_apps/62`);
  });

  it("DeleteSoftwareModal DELETEs a package by its package id", async () => {
    const { user } = premiumRender(
      <DeleteSoftwareModal
        softwareId={11}
        teamId={1}
        installerId={71}
        onExit={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("DELETE", `${BASE}/11/packages/71`);
    expect(requests[0].search).toBe("?fleet_id=1");
  });

  it("DeleteSoftwareModal DELETEs an App Store app by its version id", async () => {
    const { user } = premiumRender(
      <DeleteSoftwareModal
        softwareId={11}
        teamId={1}
        versionId={72}
        onExit={jest.fn()}
        onSuccess={jest.fn()}
      />
    );
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => expect(requests).toHaveLength(1));
    expectOnlyRequest("DELETE", `${BASE}/11/app_store_apps/72`);
    expect(requests[0].search).toBe("?fleet_id=1");
  });
});
