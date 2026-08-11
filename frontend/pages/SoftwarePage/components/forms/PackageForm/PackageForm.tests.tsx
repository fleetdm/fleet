import React from "react";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createCustomRenderer } from "test/test-utils";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";

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

const TARGET_BANNER_COPY = /If multiple packages of the same software target the same host, Fleet will install the one that was added first\./i;

describe("PackageForm", () => {
  describe("Target section on the single-package Add flow", () => {
    it("hides the Target section before a file is selected", () => {
      renderForm();
      // Target selector and its info banner should be absent until upload.
      expect(screen.queryByText(TARGET_BANNER_COPY)).not.toBeInTheDocument();
      expect(screen.queryByLabelText("All hosts")).not.toBeInTheDocument();
      expect(screen.queryByLabelText("Custom")).not.toBeInTheDocument();
      expect(screen.queryByText("Self-service")).not.toBeInTheDocument();
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
      expect(screen.getByText("Self-service")).toBeInTheDocument();
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
});
