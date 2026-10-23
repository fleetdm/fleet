import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { internationalTimeOnlyFormat } from "utilities/helpers";

import SoftwareNameCell from "./SoftwareNameCell";

const mockRouter = createMockRouter();
const defaultProps = {
  name: "Fleet Desktop",
  source: "fleet",
  router: mockRouter,
  path: "/software/1",
};

describe("SoftwareNameCell icon rendering", () => {
  // 2 "No installer" tests
  it("does not show icon when no installer (Software Title page)", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<SoftwareNameCell {...defaultProps} />);
    expect(screen.queryByTestId("install-icon")).toBeNull();
    expect(screen.queryByTestId("user-icon")).toBeNull();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  it("does not show icon when no installer (Host Inventory)", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<SoftwareNameCell {...defaultProps} pageContext="hostDetails" />);
    expect(screen.queryByTestId("install-icon")).toBeNull();
    expect(screen.queryByTestId("user-icon")).toBeNull();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  // Skip testing no installer + hostDetailsLibrary pageContext because that can never happen

  // 3 "has installer" tests
  it("shows install icon for manual installer (Software Title page)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<SoftwareNameCell {...defaultProps} hasInstaller />);
    const icon = screen.getByTestId("install-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(
        /Software can be installed on the host details page/i
      )
    ).toBeInTheDocument();
  });

  it("shows install icon for manual installer (Host Inventory)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        pageContext="hostDetails"
      />
    );
    const icon = screen.getByTestId("install-icon");
    await userEvent.hover(icon);
    expect(await screen.findByText(/on the Library tab/i)).toBeInTheDocument();
  });

  it("does not show install icon for manual installer (Host Library) as every software on that page will have an installer", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        pageContext="hostDetailsLibrary"
      />
    );
    expect(screen.queryByTestId("install-icon")).toBeNull();
  });

  // 3 "self service installer" tests
  it("shows user icon for self-service software (Software Title page)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(<SoftwareNameCell {...defaultProps} hasInstaller isSelfService />);
    const icon = screen.getByTestId("user-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/End users can install from/i)
    ).toBeInTheDocument();
  });

  it("shows user icon for self-service software (Host Inventory)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        pageContext="hostDetails"
      />
    );
    const icon = screen.getByTestId("user-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/End users can install from/i)
    ).toBeInTheDocument();
  });

  it("shows user icon for self-service software (Host Library)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        pageContext="hostDetailsLibrary"
      />
    );
    const icon = screen.getByTestId("user-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/End users can install from/i)
    ).toBeInTheDocument();
  });

  // 3 "auto installer" tests
  it("shows refresh icon for auto-install software (Software Title page)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        automaticInstallPoliciesCount={2}
      />
    );
    const icon = screen.getByTestId("refresh-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/2 policies trigger install./i)
    ).toBeInTheDocument();
  });

  it("shows refresh icon for auto-install software (Host Inventory)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        automaticInstallPoliciesCount={3}
        pageContext="hostDetails"
      />
    );
    const icon = screen.getByTestId("refresh-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/3 policies trigger install./i)
    ).toBeInTheDocument();
  });

  it("shows refresh icon for auto-install software (Host Library)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        automaticInstallPoliciesCount={1}
        pageContext="hostDetailsLibrary"
      />
    );
    const icon = screen.getByTestId("refresh-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/A policy triggers install./i)
    ).toBeInTheDocument();
  });

  // 3 "self service + auto installer" tests
  it("shows automatic-self-service icon for self-service + auto-install (Software Title page)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        automaticInstallPoliciesCount={2}
      />
    );
    const icon = screen.getByTestId("automatic-self-service-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/2 policies trigger install./i)
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/End users can install/i)
    ).toBeInTheDocument();
  });

  it("shows automatic-self-service icon for self-service + auto-install (Host Inventory)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        automaticInstallPoliciesCount={2}
        pageContext="hostDetails"
      />
    );
    const icon = screen.getByTestId("automatic-self-service-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/2 policies trigger install./i)
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/End users can install/i)
    ).toBeInTheDocument();
  });

  it("shows automatic-self-service icon for self-service + auto-install (Host Library)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        automaticInstallPoliciesCount={2}
        pageContext="hostDetailsLibrary"
      />
    );
    const icon = screen.getByTestId("automatic-self-service-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/2 policies trigger install./i)
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/End users can install/i)
    ).toBeInTheDocument();
  });

  // VPP auto-update folds into the automatic icon family: same visual as
  // policy-triggered auto-install, with an added tooltip line for the window.
  it("shows the refresh icon with a window tooltip when autoUpdateEnabled is true (no policies, no self-service)", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isIosOrIpadosApp
        isAppStoreApp
        autoUpdateEnabled
        autoUpdateWindowStart="02:00"
        autoUpdateWindowEnd="04:00"
      />
    );
    const icon = screen.getByTestId("refresh-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(
        new RegExp(
          `Auto updates between ${internationalTimeOnlyFormat(
            "02:00"
          )} and ${internationalTimeOnlyFormat(
            "04:00"
          )} \\(host local time\\)\\.`,
          "i"
        )
      )
    ).toBeInTheDocument();
  });

  it("shows the composite automatic-self-service icon with a double-barrel tooltip when self-service + auto-update", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        isIosOrIpadosApp
        isAppStoreApp
        autoUpdateEnabled
        autoUpdateWindowStart="02:00"
        autoUpdateWindowEnd="04:00"
      />
    );
    const icon = screen.getByTestId("automatic-self-service-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(
        new RegExp(
          `Auto updates between ${internationalTimeOnlyFormat(
            "02:00"
          )} and ${internationalTimeOnlyFormat(
            "04:00"
          )} \\(host local time\\)\\.`,
          "i"
        )
      )
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/End users can install/i)
    ).toBeInTheDocument();
  });

  it("stacks all three tooltip lines when policies + self-service + auto-update all apply", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        isIosOrIpadosApp
        isAppStoreApp
        automaticInstallPoliciesCount={2}
        autoUpdateEnabled
        autoUpdateWindowStart="02:00"
        autoUpdateWindowEnd="04:00"
      />
    );
    const icon = screen.getByTestId("automatic-self-service-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/2 policies trigger install\./i)
    ).toBeInTheDocument();
    expect(
      await screen.findByText(
        new RegExp(
          `Auto updates between ${internationalTimeOnlyFormat(
            "02:00"
          )} and ${internationalTimeOnlyFormat(
            "04:00"
          )} \\(host local time\\)\\.`,
          "i"
        )
      )
    ).toBeInTheDocument();
    expect(
      await screen.findByText(/End users can install/i)
    ).toBeInTheDocument();
  });

  it("keeps the self-service icon when autoUpdateEnabled is false (unchanged path)", () => {
    const render = createCustomRenderer({ withBackendMock: true });
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasInstaller
        isSelfService
        autoUpdateEnabled={false}
      />
    );
    // No refresh, no composite — just the user icon.
    expect(screen.getByTestId("user-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  describe("configuration icon with version-name tooltip", () => {
    it("renders the settings icon when deliveredVersionName is set", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <SoftwareNameCell
          {...defaultProps}
          hasInstaller
          isIosOrIpadosApp
          isAppStoreApp
          deliveredVersionName="Production"
        />
      );
      expect(screen.getByTestId("settings-icon")).toBeInTheDocument();
    });

    it("shows the version name in the tooltip on hover", async () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <SoftwareNameCell
          {...defaultProps}
          hasInstaller
          isIosOrIpadosApp
          isAppStoreApp
          deliveredVersionName="Production"
        />
      );
      await userEvent.hover(screen.getByTestId("settings-icon"));
      expect(await screen.findByText("Production")).toBeInTheDocument();
    });

    it("omits the icon when deliveredVersionName is unset", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <SoftwareNameCell
          {...defaultProps}
          hasInstaller
          isIosOrIpadosApp
          isAppStoreApp
        />
      );
      expect(screen.queryByTestId("settings-icon")).toBeNull();
    });

    it("renders alongside the install icon without replacing it", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <SoftwareNameCell
          {...defaultProps}
          hasInstaller
          isSelfService
          isIosOrIpadosApp
          isAppStoreApp
          deliveredVersionName="Beta"
        />
      );
      expect(screen.getByTestId("user-icon")).toBeInTheDocument();
      expect(screen.getByTestId("settings-icon")).toBeInTheDocument();
    });

    // Keyboard + screen-reader users can't trigger a mouse-hover tooltip, so
    // the version name is exposed via role=img + aria-label on the badge
    // wrapper. The parent LinkCell anchor is focusable; focusing it surfaces
    // the aria-label through the accessible name computation.
    it("exposes the version name as accessible text (no mouse needed)", () => {
      const render = createCustomRenderer({ withBackendMock: true });
      render(
        <SoftwareNameCell
          {...defaultProps}
          hasInstaller
          isIosOrIpadosApp
          isAppStoreApp
          deliveredVersionName="Production"
        />
      );
      expect(
        screen.getByRole("img", {
          name: /Managed configuration delivered: Production/i,
        })
      ).toBeInTheDocument();
    });
  });
});
