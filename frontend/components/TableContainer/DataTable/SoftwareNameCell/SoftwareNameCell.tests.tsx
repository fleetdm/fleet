import React from "react";
import userEvent from "@testing-library/user-event";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import SoftwareNameCell from "./SoftwareNameCell";

const mockRouter = createMockRouter();
const defaultProps = {
  name: "Fleet Desktop",
  source: "fleet",
  router: mockRouter,
  path: "/software/1",
};

describe("SoftwareNameCell icon rendering", () => {
  it("does not show icon when no installer, no self-service, no auto install (Software Title page)", () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} />);
    expect(screen.queryByTestId("install-icon")).toBeNull();
    expect(screen.queryByTestId("user-icon")).toBeNull();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  it("does not show icon when no installer, no self-service, no auto install (Host Inventory)", () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} pageContext="hostDetails" />);
    expect(screen.queryByTestId("install-icon")).toBeNull();
    expect(screen.queryByTestId("user-icon")).toBeNull();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  it("does not show icon when no self-service, no auto install since installer icon is redundant because every item on host library is an installer (Host Library)", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell {...defaultProps} pageContext="hostDetailsLibrary" />
    );
    expect(screen.queryByTestId("install-icon")).toBeNull();
    expect(screen.queryByTestId("user-icon")).toBeNull();
    expect(screen.queryByTestId("refresh-icon")).toBeNull();
    expect(screen.queryByTestId("automatic-self-service-icon")).toBeNull();
  });

  it("shows user icon for self-service software (Software Title page)", async () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} hasPackage isSelfService />);
    const icon = screen.getByTestId("user-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/End users can install from/i)
    ).toBeInTheDocument();
  });

  it("shows user icon for self-service software (Host Inventory)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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

  it("shows refresh icon for auto-install software (Software Title page)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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

  it("shows automatic-self-service icon for self-service + auto-install (Software Title page)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
      await screen.findByText(/End users can reinstall/i)
    ).toBeInTheDocument();
  });

  it("shows automatic-self-service icon for self-service + auto-install (Host Inventory)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
      await screen.findByText(/End users can reinstall/i)
    ).toBeInTheDocument();
  });

  it("shows automatic-self-service icon for self-service + auto-install (Host Library)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
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
      await screen.findByText(/End users can reinstall/i)
    ).toBeInTheDocument();
  });

  it("shows install icon for manual installer (Software Title page)", async () => {
    const render = createCustomRenderer();
    render(<SoftwareNameCell {...defaultProps} hasPackage />);
    const icon = screen.getByTestId("install-icon");
    await userEvent.hover(icon);
    expect(
      await screen.findByText(/Software can be installed on host details page/i)
    ).toBeInTheDocument();
  });

  it("shows install icon for manual installer (Host Inventory)", async () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
        pageContext="hostDetails"
      />
    );
    const icon = screen.getByTestId("install-icon");
    await userEvent.hover(icon);
    expect(await screen.findByText(/on the Library tab/i)).toBeInTheDocument();
  });

  it("does not show install icon for manual installer (Host Library, redundant)", () => {
    const render = createCustomRenderer();
    render(
      <SoftwareNameCell
        {...defaultProps}
        hasPackage
        pageContext="hostDetailsLibrary"
      />
    );
    expect(screen.queryByTestId("install-icon")).toBeNull();
  });
});
