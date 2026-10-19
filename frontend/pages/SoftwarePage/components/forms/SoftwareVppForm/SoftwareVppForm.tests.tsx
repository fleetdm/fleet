import { screen } from "@testing-library/react";
import React from "react";

import { createMockVppApp } from "__mocks__/appleMdm";
import createMockConfig from "__mocks__/configMock";
import { createMockAppStoreApp } from "__mocks__/softwareMock";
import { createCustomRenderer } from "test/test-utils";

import SoftwareVppForm from "./SoftwareVppForm";

describe("SoftwareVppForm", () => {
  it("shows Self service after selecting an app to add", async () => {
    const render = createCustomRenderer({ withBackendMock: true });
    const { user } = render(
      <SoftwareVppForm
        labels={[]}
        vppApps={[createMockVppApp()]}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
        onClickPreviewEndUserExperience={jest.fn()}
        teamId={1}
      />
    );

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Test App/ }));

    expect(screen.getByText("Self service")).toBeInTheDocument();
  });

  it("disables form controls on the Edit path in GitOps mode so keyboard activation can't mutate state behind the CSS lock", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
          config: createMockConfig({
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "https://example.com/repo",
            },
          }),
        },
      },
    });
    render(
      <SoftwareVppForm
        labels={[]}
        softwareVppForEdit={createMockAppStoreApp({
          platform: "darwin",
          self_service: true,
        })}
        onSubmit={jest.fn()}
        onCancel={jest.fn()}
        onClickPreviewEndUserExperience={jest.fn()}
        teamId={1}
      />
    );

    expect(screen.getByRole("switch")).toBeDisabled();
    screen
      .getAllByRole("radio")
      .forEach((radio) => expect(radio).toBeDisabled());
  });
});
