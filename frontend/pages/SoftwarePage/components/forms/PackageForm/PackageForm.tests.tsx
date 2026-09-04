import React from "react";
import { render as renderComponent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createCustomRenderer } from "test/test-utils";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import { AppContext, IAppContext } from "context/app";

import PackageForm from "./PackageForm";

const BASE_PROPS = {
  labels: [],
  onCancel: jest.fn(),
  onSubmit: jest.fn(),
  onClickPreviewEndUserExperience: jest.fn(),
};

const renderForm = (
  overrides: Partial<React.ComponentProps<typeof PackageForm>> = {},
  config?: Partial<IConfig>
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        config,
      },
    },
  });
  return render(<PackageForm {...BASE_PROPS} {...overrides} />);
};

const ONE_GIB = 1024 * 1024 * 1024;

// The form reads File.size, so fake the size rather than allocating a real
// multi-gigabyte buffer.
const selectFileOfSize = async (container: HTMLElement, size: number) => {
  const file = new File(["installer"], "test.pkg");
  Object.defineProperty(file, "size", { value: size });
  const input = container.querySelector("#upload-file") as HTMLInputElement;
  await userEvent.upload(input, file);
};

const selectFileNamed = async (container: HTMLElement, name: string) => {
  const file = new File(["installer"], name);
  const input = container.querySelector("#upload-file") as HTMLInputElement;
  // Bypass the input's `accept` attribute so we can exercise the client-side
  // extension-guard path (`getDefaultInstallScript`/`Uninstall`) for files a
  // drag-and-drop or a browser lenient with MIME sniffing would let through.
  await userEvent.upload(input, file, { applyAccept: false });
};

const TARGET_BANNER_COPY = /If multiple packages of the same software target the same host, Fleet will install the one that was added first\./i;

describe("PackageForm", () => {
  describe("Target section on the single-package Add flow", () => {
    it("hides the Target section before a file is selected", () => {
      renderForm();
      // Target selector and its info banner should be absent until upload.
      expect(screen.queryByText(TARGET_BANNER_COPY)).not.toBeInTheDocument();
      expect(screen.queryByLabelText("All hosts")).not.toBeInTheDocument();
      expect(screen.queryByLabelText("Custom")).not.toBeInTheDocument();
      expect(screen.queryByText("Self service")).not.toBeInTheDocument();
    });

    it("renders the Target section with the first-added banner once a file is selected", () => {
      // `defaultSoftware` seeds initialFormData.software (the form casts it to
      // File internally), which is the same signal the Add flow raises when
      // the user picks a file. Uses the ISoftwarePackage mock to satisfy the
      // prop's declared type.
      renderForm({ defaultSoftware: createMockSoftwarePackage() });
      expect(screen.getByText(TARGET_BANNER_COPY)).toBeInTheDocument();
      expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      expect(screen.getByLabelText("Custom")).toBeInTheDocument();
      expect(screen.getByText("Self service")).toBeInTheDocument();
    });

    it("omits the first-added banner on the Edit flow", () => {
      renderForm({
        isEditingSoftware: true,
        defaultSoftware: createMockSoftwarePackage(),
      });
      // Target selector is present on Edit, but the banner is not — the
      // install-order copy only applies to Add flows.
      expect(screen.queryByText(TARGET_BANNER_COPY)).not.toBeInTheDocument();
      expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
      expect(screen.getByLabelText("Custom")).toBeInTheDocument();
    });
  });

  describe("Advanced options section", () => {
    it("reveals install/uninstall scripts when clicked for a .msix package", async () => {
      renderForm({
        isEditingSoftware: true,
        defaultSoftware: createMockSoftwarePackage({ name: "Claude.msix" }),
        defaultInstallScript: "Add-AppxProvisionedPackage -Online",
        defaultUninstallScript: "Remove-AppxProvisionedPackage -Online",
      });

      // Scripts hidden until the reveal button is clicked.
      expect(screen.queryByText("Install script")).not.toBeInTheDocument();

      await userEvent.click(
        screen.getByRole("button", { name: /Advanced options/i })
      );

      expect(screen.getByText("Install script")).toBeInTheDocument();
      expect(screen.getByText("Uninstall script")).toBeInTheDocument();
    });
  });

  describe("Maximum package size", () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    it("rejects a package over the configured maximum before uploading", async () => {
      const errorSpy = jest.spyOn(notify, "error");
      const { container } = renderForm(
        {},
        { max_software_package_size: ONE_GIB }
      );

      await selectFileOfSize(container, ONE_GIB + 1);

      expect(errorSpy).toHaveBeenCalledWith(
        "Couldn't add. The maximum file size is 1GiB."
      );
      // The rejected file never reaches form state, so the Target section
      // stays hidden.
      expect(screen.queryByLabelText("All hosts")).not.toBeInTheDocument();
    });

    it("rejects any package when the limit is zero", async () => {
      // A zero limit is a real setting, not a missing one, and the server
      // refuses every upload under it.
      const errorSpy = jest.spyOn(notify, "error");
      const { container } = renderForm({}, { max_software_package_size: 0 });

      await selectFileOfSize(container, 1);

      expect(errorSpy).toHaveBeenCalledWith(
        "Couldn't add. The maximum file size is 0B."
      );
    });

    it("accepts a package at the configured maximum", async () => {
      const errorSpy = jest.spyOn(notify, "error");
      const { container } = renderForm(
        {},
        { max_software_package_size: ONE_GIB }
      );

      await selectFileOfSize(container, ONE_GIB);

      expect(errorSpy).not.toHaveBeenCalled();
      expect(screen.getByLabelText("All hosts")).toBeInTheDocument();
    });
  });

  describe("Unsupported file extension on add", () => {
    afterEach(() => {
      jest.restoreAllMocks();
    });

    // Regression: the raw Error was passed as the toast response, so the
    // main line rendered "Error: unsupported file extension: dmg" and the
    // expandable panel opened on `{}` (Error's own props are non-enumerable).
    it("shows a friendly toast with the reason in the response payload when the install-script derivation throws", async () => {
      const errorSpy = jest.spyOn(notify, "error");
      const { container } = renderForm();

      await selectFileNamed(container, "test.dmg");

      expect(errorSpy).toHaveBeenCalledWith("Couldn't add.", {
        response: {
          data: { message: "unsupported file extension: dmg" },
        },
      });
    });

    // .zip passes the install switch (returns "") but trips the uninstall
    // switch's default, so this covers the second catch block.
    it("shows a friendly toast with the reason in the response payload when the uninstall-script derivation throws", async () => {
      const errorSpy = jest.spyOn(notify, "error");
      const { container } = renderForm();

      await selectFileNamed(container, "test.zip");

      expect(errorSpy).toHaveBeenCalledWith("Couldn't add.", {
        response: {
          data: { message: "unsupported file extension: zip" },
        },
      });
    });
  });

  describe("GitOps mode with a preselected Custom target", () => {
    const gitOpsConfig = {
      gitops: {
        gitops_mode_enabled: true,
        repository_url: "https://example.com/repo",
        exceptions: { labels: false, software: false, secrets: false },
      },
    } as Partial<IConfig>;

    // `config` is fetched asynchronously and App renders its children before it
    // resolves, so GitOps mode is often unknown on the first render. Rendering
    // through AppContext directly is what lets the config value change after
    // mount, which `createCustomRenderer` fixes in place.
    const renderWithConfig = (config: Partial<IConfig> | null) => (
      <AppContext.Provider
        value={({ config, isPremiumTier: true } as unknown) as IAppContext}
      >
        <PackageForm
          {...BASE_PROPS}
          multiPackageContext
          initialTargetType="Custom"
        />
      </AppContext.Provider>
    );

    it("enables Save even though the target selector is hidden", async () => {
      const { container } = renderForm(
        { multiPackageContext: true, initialTargetType: "Custom" },
        gitOpsConfig
      );

      await selectFileOfSize(container, 1024);

      // GitOps mode hides the selector, so the user can't pick labels. A
      // preselected "Custom" would leave Save disabled with nothing on screen
      // to explain why.
      expect(screen.queryByLabelText("Custom")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).not.toBeDisabled();
    });

    it("normalizes the target when config arrives after mount", async () => {
      const { container, rerender } = renderComponent(renderWithConfig(null));
      rerender(renderWithConfig(gitOpsConfig));

      await selectFileOfSize(container, 1024);

      expect(screen.queryByLabelText("Custom")).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).not.toBeDisabled();
    });

    it("enables Save when the file is chosen before config arrives", async () => {
      // The reverse ordering of the previous test: validation is computed on
      // file select against the still-preselected "Custom", so normalizing the
      // target afterwards has to refresh validation too.
      const { container, rerender } = renderComponent(renderWithConfig(null));

      await selectFileOfSize(container, 1024);
      rerender(renderWithConfig(gitOpsConfig));

      expect(screen.getByRole("button", { name: "Save" })).not.toBeDisabled();
    });

    it("keeps requiring labels for a Custom target when GitOps mode is off", async () => {
      const { container } = renderForm({
        multiPackageContext: true,
        initialTargetType: "Custom",
      });

      await selectFileOfSize(container, 1024);

      // The selector is visible here, so the preselection stands and the
      // existing "pick at least one label" rule still applies.
      expect(screen.getByLabelText("Custom")).toBeChecked();
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
    });
  });
});
