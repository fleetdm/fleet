import React from "react";
import { screen } from "@testing-library/react";
import { noop } from "lodash";
import { renderWithAppContext, createCustomRenderer } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import AddHostsModal from "./AddHostsModal";

const ENROLL_SECRET = "abcdefg12345678";

describe("AddHostsModal", () => {
  it("renders loading state", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <AddHostsModal isAnyTeamSelected={false} isLoading onCancel={noop} />
    );
    const loadingSpinner = screen.getByTestId("spinner");
    expect(loadingSpinner).toBeVisible();
  });
  it("renders platform tabs", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    const { user } = render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );

    await user.click(screen.getByRole("tab", { name: "macOS" }));
    const macOSText = screen.getByText("--type=pkg");
    expect(macOSText).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    const windowsText = screen.getByText("--type=msi");
    expect(windowsText).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Linux (RPM)" }));
    const linuxRPMText = screen.getByText("--type=rpm");
    expect(linuxRPMText).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Linux (deb)" }));
    const linuxDebText = screen.getByText("--type=deb");
    expect(linuxDebText).toBeInTheDocument();

    // await user.click(screen.getByRole("tab", { name: "Advanced" }));
    // const advancedText = screen.getByText("--type=YOUR_TYPE");
    // expect(advancedText).toBeInTheDocument();
  });
  it("renders installer with secret", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <AddHostsModal
        isAnyTeamSelected
        enrollSecret={ENROLL_SECRET}
        isLoading={false}
        onCancel={noop}
      />
    );
    const text = screen.getByText(ENROLL_SECRET);

    expect(text).toBeInTheDocument();
  });
  it("renders no enroll secret cta", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPreviewMode: false,
          config: createMockConfig(),
        },
      },
    });

    render(
      <AddHostsModal
        isAnyTeamSelected={false}
        currentTeamName={"Apples"}
        isLoading={false}
        onCancel={noop}
        openEnrollSecretModal={noop}
      />
    );

    const text = screen.getByText("Something's gone wrong.");
    const ctaButton = screen.getByRole("button", {
      name: "Manage enroll secrets",
    });

    expect(text).toBeInTheDocument();
    expect(ctaButton).toBeEnabled();
  });

  describe("user is in sandbox mode", () => {
    const contextValue = {
      isSandboxMode: true,
    };

    it("download is disabled until a platform is selected", async () => {
      // TODO:
      // Need backend mock for certificate
      // const render = createCustomRenderer({
      //   withBackendMock: true,
      // });

      // Need app context for isSandboxMod
      renderWithAppContext(
        <AddHostsModal
          isAnyTeamSelected={false}
          isLoading={false}
          enrollSecret={ENROLL_SECRET}
          onCancel={noop}
        />,
        { contextValue }
      );

      // Need regular render (not with app context for user click)
      const text = screen.getByText("Which platform");
      const windowsText = screen.getByText("Windows");
      const downloadButton = screen.getByRole("button", {
        name: /Download installer/i,
      });

      expect(text).toBeInTheDocument();
      expect(screen.getByRole(downloadButton)).not.toBeEnabled();

      // TODO: Allow "user" click with app context render
      // await user.click(windowsText);

      // expect(screen.getByRole(downloadButton)).toBeEnabled();
    });
  });
});
