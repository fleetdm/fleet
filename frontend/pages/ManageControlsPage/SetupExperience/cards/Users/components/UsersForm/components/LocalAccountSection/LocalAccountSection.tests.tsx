import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { EndUserLocalAccountType } from "interfaces/mdm";
import { createCustomRenderer } from "test/test-utils";

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
      enableManagedLocalAccountWindows: false,
    },
    onLocalAccountTypeChange: onLocalAccountTypeChangeMock,
    onEnableManagedLocalAccountChange: onEnableManagedLocalAccountChangeMock,
    isMacMdmEnabledAndConfigured: true,
  };

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  it("renders the sub-section headings", () => {
    render(<LocalAccountSection {...defaultProps} />);
    expect(screen.getByText("End user account")).toBeInTheDocument();
    expect(screen.getByText("Managed account")).toBeInTheDocument();
  });

  it("renders the managed local account checkbox with help text", () => {
    render(<LocalAccountSection {...defaultProps} />);
    expect(
      screen.getByRole("checkbox", { name: "Create hidden admin" })
    ).toBeInTheDocument();
    expect(
      screen.getByText("A hidden local admin for remote troubleshooting.")
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

  describe("GitOps mode tooltips", () => {
    const gitopsRender = createCustomRenderer({
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
    const tooltipWrapperOf = (el: HTMLElement) =>
      el.closest(".gitops-mode-tooltip-wrapper");

    it("gives the account options and the managed account checkbox separate tooltips", () => {
      gitopsRender(<LocalAccountSection {...defaultProps} />);

      const optionsWrapper = tooltipWrapperOf(
        screen.getByRole("radio", { name: "Admin" })
      );
      const checkboxWrapper = tooltipWrapperOf(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      );
      expect(optionsWrapper).not.toBeNull();
      expect(checkboxWrapper).not.toBeNull();
      expect(checkboxWrapper).not.toBe(optionsWrapper);
      expect(
        tooltipWrapperOf(
          screen.getByRole("radio", { name: "Skip (no account)" })
        )
      ).toBe(optionsWrapper);

      expect(tooltipWrapperOf(screen.getByText("End user account"))).toBeNull();
      expect(tooltipWrapperOf(screen.getByText("Managed account"))).toBeNull();
    });

    it("shows the tooltip when hovering an option's help text", async () => {
      const { user } = gitopsRender(<LocalAccountSection {...defaultProps} />);

      await user.hover(
        screen.getByText(
          "End user can install apps and change their own settings only."
        )
      );
      await waitFor(() => {
        expect(screen.getAllByText(/GitOps mode enabled/)).toHaveLength(1);
      });
    });

    it("shows the tooltip when hovering the managed account checkbox", async () => {
      const { user } = gitopsRender(<LocalAccountSection {...defaultProps} />);

      await user.hover(
        screen.getByRole("checkbox", { name: "Create hidden admin" })
      );
      await waitFor(() => {
        expect(screen.getAllByText(/GitOps mode enabled/)).toHaveLength(1);
      });
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
