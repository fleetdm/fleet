import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockSoftwarePackage } from "__mocks__/softwareMock";

import PackageForm from "./PackageForm";

const BASE_PROPS = {
  labels: [],
  onCancel: jest.fn(),
  onSubmit: jest.fn(),
  onClickPreviewEndUserExperience: jest.fn(),
};

const renderForm = (
  overrides: Partial<React.ComponentProps<typeof PackageForm>> = {}
) => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
      },
    },
  });
  return render(<PackageForm {...BASE_PROPS} {...overrides} />);
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
});
