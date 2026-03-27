import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import createMockConfig from "__mocks__/configMock";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import ChangeManagement from "./ChangeManagement";

const configUrl = baseUrl("/config");

// New-install default: only enroll secrets exception is enabled
const newInstallGitopsConfig = {
  gitops_mode_enabled: false,
  repository_url: "",
  exceptions: { labels: false, software: false, secrets: true },
};

// Pre-existing (migrated) instance: labels and enroll secrets exceptions are enabled
const migratedGitopsConfig = {
  gitops_mode_enabled: false,
  repository_url: "",
  exceptions: { labels: true, software: false, secrets: true },
};

const createGetConfigHandler = (
  gitopsOverrides = newInstallGitopsConfig
) =>
  http.get(configUrl, () =>
    HttpResponse.json(createMockConfig({ gitops: gitopsOverrides }))
  );

// The Checkbox component renders a div[role="checkbox"] with aria-label equal to
// the `name` prop, so we query by the name prop values used in ChangeManagement.
const LABELS_CHECKBOX_NAME = "exceptLabels";
const SOFTWARE_CHECKBOX_NAME = "exceptSoftware";
const SECRETS_CHECKBOX_NAME = "exceptSecrets";
const GITOPS_MODE_CHECKBOX_NAME = "gitOpsModeEnabled";

describe("ChangeManagement", () => {
  describe("Premium tier", () => {
    it("shows premium feature message when not premium tier", () => {
      mockServer.use(createGetConfigHandler());

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: false, setConfig: jest.fn() } },
      });

      render(<ChangeManagement />);

      expect(
        screen.getByText(/This feature is included in Fleet Premium/i)
      ).toBeInTheDocument();
    });
  });

  describe("New install default state", () => {
    it("renders only Enroll secrets checked for a new install", async () => {
      mockServer.use(createGetConfigHandler(newInstallGitopsConfig));

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, setConfig: jest.fn() } },
      });

      render(<ChangeManagement />);

      // Wait for config to load and form to appear
      const enrollSecretsCheckbox = await screen.findByRole("checkbox", {
        name: SECRETS_CHECKBOX_NAME,
      });

      const labelsCheckbox = screen.getByRole("checkbox", {
        name: LABELS_CHECKBOX_NAME,
      });
      const softwareCheckbox = screen.getByRole("checkbox", {
        name: SOFTWARE_CHECKBOX_NAME,
      });

      expect(labelsCheckbox).not.toBeChecked();
      expect(softwareCheckbox).not.toBeChecked();
      expect(enrollSecretsCheckbox).toBeChecked();
    });
  });

  describe("Migrated instance state", () => {
    it("renders Labels and Enroll secrets checked for a migrated instance", async () => {
      mockServer.use(createGetConfigHandler(migratedGitopsConfig));

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, setConfig: jest.fn() } },
      });

      render(<ChangeManagement />);

      // Wait for config to load and form to appear
      const labelsCheckbox = await screen.findByRole("checkbox", {
        name: LABELS_CHECKBOX_NAME,
      });

      const softwareCheckbox = screen.getByRole("checkbox", {
        name: SOFTWARE_CHECKBOX_NAME,
      });
      const enrollSecretsCheckbox = screen.getByRole("checkbox", {
        name: SECRETS_CHECKBOX_NAME,
      });

      expect(labelsCheckbox).toBeChecked();
      expect(softwareCheckbox).not.toBeChecked();
      expect(enrollSecretsCheckbox).toBeChecked();
    });
  });

  describe("Form submission", () => {
    it("sends the correct exceptions payload to configAPI.update on save", async () => {
      mockServer.use(createGetConfigHandler(newInstallGitopsConfig));

      // Capture the request body sent via PATCH
      let capturedBody: any;
      mockServer.use(
        http.patch(configUrl, async ({ request }) => {
          capturedBody = await request.json();
          return HttpResponse.json(
            createMockConfig({
              gitops: {
                gitops_mode_enabled: false,
                repository_url: "",
                exceptions: { labels: true, software: false, secrets: true },
              },
            })
          );
        })
      );

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, setConfig: jest.fn() } },
      });

      const { user } = render(<ChangeManagement />);

      // Wait for config to load
      const labelsCheckbox = await screen.findByRole("checkbox", {
        name: LABELS_CHECKBOX_NAME,
      });

      // Toggle Labels on
      await user.click(labelsCheckbox);

      // Submit the form
      const saveButton = screen.getByRole("button", { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(capturedBody).toMatchObject({
          gitops: {
            gitops_mode_enabled: false,
            repository_url: "",
            exceptions: { labels: true, software: false, secrets: true },
          },
        });
      });
    });

    it("sends correct payload when GitOps mode is enabled with a repo URL", async () => {
      mockServer.use(createGetConfigHandler(newInstallGitopsConfig));

      let capturedBody: any;
      mockServer.use(
        http.patch(configUrl, async ({ request }) => {
          capturedBody = await request.json();
          return HttpResponse.json(
            createMockConfig({
              gitops: {
                gitops_mode_enabled: true,
                repository_url: "https://github.com/org/repo",
                exceptions: { labels: false, software: false, secrets: true },
              },
            })
          );
        })
      );

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, setConfig: jest.fn() } },
      });

      const { user } = render(<ChangeManagement />);

      // Wait for form to load
      await screen.findByRole("checkbox", { name: SECRETS_CHECKBOX_NAME });

      // Enable GitOps mode
      const gitOpsModeCheckbox = screen.getByRole("checkbox", {
        name: GITOPS_MODE_CHECKBOX_NAME,
      });
      await user.click(gitOpsModeCheckbox);

      // Enter a valid repository URL
      const repoUrlInput = screen.getByRole("textbox");
      await user.type(repoUrlInput, "https://github.com/org/repo");

      // Submit the form
      const saveButton = screen.getByRole("button", { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(capturedBody).toMatchObject({
          gitops: {
            gitops_mode_enabled: true,
            repository_url: "https://github.com/org/repo",
            exceptions: { labels: false, software: false, secrets: true },
          },
        });
      });
    });
  });

  describe("Form validation", () => {
    it("shows an error when GitOps mode is enabled but no repository URL is provided", async () => {
      mockServer.use(createGetConfigHandler(newInstallGitopsConfig));

      const render = createCustomRenderer({
        withBackendMock: true,
        context: { app: { isPremiumTier: true, setConfig: jest.fn() } },
      });

      const { user } = render(<ChangeManagement />);

      // Wait for form to load
      await screen.findByRole("checkbox", { name: SECRETS_CHECKBOX_NAME });

      // Enable GitOps mode (without entering a URL)
      const gitOpsModeCheckbox = screen.getByRole("checkbox", {
        name: GITOPS_MODE_CHECKBOX_NAME,
      });
      await user.click(gitOpsModeCheckbox);

      // Submit
      const saveButton = screen.getByRole("button", { name: /save/i });
      await user.click(saveButton);

      expect(
        await screen.findByText(
          /git repository url is required when gitops mode is enabled/i
        )
      ).toBeInTheDocument();
    });
  });
});
