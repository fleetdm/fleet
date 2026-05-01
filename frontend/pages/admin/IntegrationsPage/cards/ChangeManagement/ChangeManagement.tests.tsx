import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import createMockConfig from "__mocks__/configMock";
import { IConfig } from "interfaces/config";

import ChangeManagement from "./ChangeManagement";

const configUrl = baseUrl("/config");

const createGetConfigHandler = (overrides?: Partial<IConfig>) => {
  return http.get(configUrl, () => {
    return HttpResponse.json(createMockConfig(overrides));
  });
};

const createPatchConfigHandler = (spy: jest.Mock) => {
  return http.patch(configUrl, async ({ request }) => {
    const body = await request.json();
    spy(body);
    // Echo back a full config with the gitops fields from the request
    return HttpResponse.json(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      createMockConfig({ gitops: (body as any).gitops })
    );
  });
};

describe("ChangeManagement", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { isPremiumTier: true, setConfig: jest.fn() },
      notification: { renderFlash: jest.fn() },
    },
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("GitOps mode checkbox", () => {
    it("is checked when API returns gitops_mode_enabled: true", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/repo",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(
          screen.getByRole("checkbox", { name: "gitOpsModeEnabled" })
        ).toHaveAttribute("aria-checked", "true");
      });
    });

    it("is unchecked when API returns gitops_mode_enabled: false", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(
          screen.getByRole("checkbox", { name: "gitOpsModeEnabled" })
        ).not.toHaveAttribute("aria-checked", "true");
      });
    });
  });

  describe("GitOps URL field", () => {
    it("populates with repository_url from API response", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/repo",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      expect(
        await screen.findByDisplayValue("https://github.com/org/repo")
      ).toBeInTheDocument();
    });

    it("is disabled when GitOps mode is off", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(screen.getByLabelText(/git repository url/i)).toBeDisabled();
      });
    });

    it("is enabled when GitOps mode is on", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/repo",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(screen.getByLabelText(/git repository url/i)).not.toBeDisabled();
      });
    });
  });

  describe("Form validation", () => {
    it("shows error when saving with GitOps mode enabled and no URL", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "",
            exceptions: { labels: false, software: false, secrets: true },
          },
        })
      );

      const { user } = render(<ChangeManagement />);

      const saveButton = await screen.findByRole("button", { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(
          screen.getByText(
            /git repository url is required when gitops mode is enabled/i
          )
        ).toBeInTheDocument();
      });
    });
  });

  describe("Exception checkboxes", () => {
    it("populates from API response", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: true, software: false, secrets: true },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(
          screen.getByRole("checkbox", { name: "exceptLabels" })
        ).toHaveAttribute("aria-checked", "true");
        expect(
          screen.getByRole("checkbox", { name: "exceptSoftware" })
        ).not.toHaveAttribute("aria-checked", "true");
        expect(
          screen.getByRole("checkbox", { name: "exceptSecrets" })
        ).toHaveAttribute("aria-checked", "true");
      });
    });

    it("reflects all false when API returns all false", async () => {
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: false, software: false, secrets: false },
          },
        })
      );

      render(<ChangeManagement />);

      await waitFor(() => {
        expect(
          screen.getByRole("checkbox", { name: "exceptLabels" })
        ).not.toHaveAttribute("aria-checked", "true");
        expect(
          screen.getByRole("checkbox", { name: "exceptSoftware" })
        ).not.toHaveAttribute("aria-checked", "true");
        expect(
          screen.getByRole("checkbox", { name: "exceptSecrets" })
        ).not.toHaveAttribute("aria-checked", "true");
      });
    });
  });

  describe("Form submission", () => {
    it("sends correct data to API on save", async () => {
      const patchSpy = jest.fn();
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: false, software: false, secrets: true },
          },
        }),
        createPatchConfigHandler(patchSpy)
      );

      const { user } = render(<ChangeManagement />);

      // Wait for form to load with API data
      await screen.findByRole("checkbox", { name: "exceptLabels" });

      // Toggle the labels exception on
      const labelsCheckbox = screen.getByRole("checkbox", {
        name: "exceptLabels",
      });
      await user.click(labelsCheckbox);

      const saveButton = screen.getByRole("button", { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(patchSpy).toHaveBeenCalledWith({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: {
              labels: true,
              software: false,
              secrets: true,
            },
          },
        });
      });
    });

    it("sends updated URL to API on save", async () => {
      const patchSpy = jest.fn();
      mockServer.use(
        createGetConfigHandler({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/repo",
            exceptions: { labels: false, software: false, secrets: true },
          },
        }),
        createPatchConfigHandler(patchSpy)
      );

      const { user } = render(<ChangeManagement />);

      const urlInput = await screen.findByDisplayValue(
        "https://github.com/org/repo"
      );
      await user.clear(urlInput);
      await user.type(urlInput, "https://github.com/org/new-repo");

      const saveButton = screen.getByRole("button", { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(patchSpy).toHaveBeenCalledWith({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/new-repo",
            exceptions: {
              labels: false,
              software: false,
              secrets: true,
            },
          },
        });
      });
    });
  });
});
