import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";
import { EndUserLocalAccountType } from "interfaces/mdm";

import LocalAccountSection from "./LocalAccountSection";

describe("LocalAccountSection", () => {
  const onLocalAccountTypeChangeMock = jest.fn();
  const onEnableManagedLocalAccountChangeMock = jest.fn();
  beforeEach(() => {
    onLocalAccountTypeChangeMock.mockClear();
    onEnableManagedLocalAccountChangeMock.mockClear();
  });

  const defaultProps = {
    formData: {
      endUserAuthEnabled: false,
      lockEndUserInfo: false,
      enableManagedLocalAccount: false,
      localAccountType: EndUserLocalAccountType.ADMIN,
    },
    onLocalAccountTypeChange: onLocalAccountTypeChangeMock,
    onEnableManagedLocalAccountChange: onEnableManagedLocalAccountChangeMock,
    isMacMdmEnabledAndConfigured: true,
  };

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the section title and subtitle", () => {
    render(<LocalAccountSection {...defaultProps} />);
    expect(screen.getByText("Local accounts")).toBeInTheDocument();
    expect(
      screen.getByText(/Currently supported for macOS hosts/)
    ).toBeInTheDocument();
  });

  it("renders the managed local account checkbox with help text", () => {
    render(<LocalAccountSection {...defaultProps} />);
    expect(
      screen.getByRole("checkbox", { name: "Create hidden admin" })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Fleet creates a user \(_fleetadmin\)/)
    ).toBeInTheDocument();
  });

  describe("active states", () => {
    it("renders managed local account checkbox as unchecked by default", () => {
      render(<LocalAccountSection {...defaultProps} />);
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).not.toBeChecked();
    });

    it("renders managed local account checkbox as checked when default is true", () => {
      render(
        <LocalAccountSection
          {...defaultProps}
          formData={{
            ...defaultProps.formData,
            enableManagedLocalAccount: true,
          }}
        />
      );
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toBeChecked();
    });

    it("managed local account is active when local account type is Admin", async () => {
      const { user } = render(<LocalAccountSection {...defaultProps} />);
      const checkbox = screen.getByRole("checkbox", {
        name: "Create hidden admin",
      });
      expect(checkbox).not.toBeDisabled();
      expect(checkbox).not.toBeChecked();
      await user.click(checkbox);
      expect(onEnableManagedLocalAccountChangeMock).toHaveBeenCalledWith(true);
    });
  });

  describe("disabled states", () => {
    it("when Apple MDM is not enabled and configured", async () => {
      const { user } = render(
        <LocalAccountSection
          {...defaultProps}
          isMacMdmEnabledAndConfigured={false}
        />
      );
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toHaveAttribute("aria-disabled", "true");

      // Expect 3 radio buttons to also be disabled
      expect(screen.getByRole("radio", { name: "Admin" })).toBeDisabled();
      expect(screen.getByRole("radio", { name: "Standard" })).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: "Skip (no account)" })
      ).toBeDisabled();

      // check tooltip is apple MDM related
      await user.hover(screen.getByRole("radio", { name: "Admin" }));
      await waitFor(() => {
        expect(
          screen.getByText(/To enable, first turn on/)
        ).toBeInTheDocument();
      });
    });

    it("does not allow toggling managed local account when Apple MDM is not configured", async () => {
      const { user } = render(
        <LocalAccountSection
          {...defaultProps}
          isMacMdmEnabledAndConfigured={false}
        />
      );

      const checkbox = screen.getByRole("checkbox", {
        name: "Create hidden admin",
      });
      await user.click(checkbox);

      expect(checkbox).not.toBeChecked();
    });

    it("when GitOps mode is enabled", async () => {
      const gitopsDisabledRender = createCustomRenderer({
        context: {
          app: {
            config: {
              gitops: {
                gitops_mode_enabled: true,
                repository_url: "https://example.com/repo.git",
              },
            },
          },
        },
      });
      const { user } = gitopsDisabledRender(
        <LocalAccountSection {...defaultProps} />
      );
      expect(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      ).toHaveAttribute("aria-disabled", "true");

      // Expect 3 radio buttons to also be disabled
      expect(screen.getByRole("radio", { name: "Admin" })).toBeDisabled();
      expect(screen.getByRole("radio", { name: "Standard" })).toBeDisabled();
      expect(
        screen.getByRole("radio", { name: "Skip (no account)" })
      ).toBeDisabled();

      // check tooltip is GitOps related
      await user.hover(screen.getByRole("radio", { name: "Admin" }));
      await waitFor(() => {
        expect(screen.getByText(/GitOps mode enabled/)).toBeInTheDocument();
      });
    });

    it.each([EndUserLocalAccountType.STANDARD, EndUserLocalAccountType.NONE])(
      "when local account type is %s disables and forces managed local account on",
      async (accountType) => {
        const { user } = render(
          <LocalAccountSection
            {...defaultProps}
            formData={{
              ...defaultProps.formData,
              localAccountType: accountType,
            }}
          />
        );
        const checkbox = screen.getByRole("checkbox", {
          name: "Create hidden admin",
        });
        expect(checkbox).toHaveAttribute("aria-disabled", "true");
        expect(checkbox).toBeChecked();
        await user.click(checkbox);
        expect(onEnableManagedLocalAccountChangeMock).not.toHaveBeenCalled();
      }
    );
  });
});
