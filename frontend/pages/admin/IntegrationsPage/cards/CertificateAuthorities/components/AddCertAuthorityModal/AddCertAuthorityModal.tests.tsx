import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { createCustomRenderer, renderWithSetup } from "test/test-utils";
import createMockConfig from "__mocks__/configMock";

import AddCertAuthorityModal from "./AddCertAuthorityModal";

describe("AddCertAuthorityModal", () => {
  it("renders the Digicert form by default", () => {
    render(<AddCertAuthorityModal onExit={noop} />);

    expect(screen.getByLabelText("Name")).toBeVisible();
    expect(screen.getByLabelText("URL")).toBeVisible();
    expect(screen.getByLabelText("API token")).toBeVisible();
    expect(screen.getByLabelText("Profile GUID")).toBeVisible();
    expect(screen.getByLabelText("Certificate common name (CN)")).toBeVisible();
    expect(screen.getByLabelText("User principal name (UPN)")).toBeVisible();
    expect(screen.getByLabelText("Certificate seat ID")).toBeVisible();
  });

  it("shows the correct form when the corresponding value in the dropdown is selected.", async () => {
    const { user } = renderWithSetup(<AddCertAuthorityModal onExit={noop} />);

    // this is selecting the custom scep option from the dropdown
    await user.click(screen.getByRole("combobox"));
    await user.click(
      screen.getByRole("option", {
        name: "Custom (SCEP: Simple Certificate Enrollment Protocol)",
      })
    );

    expect(screen.getByLabelText("Name")).toBeVisible();
    expect(screen.getByLabelText("SCEP URL")).toBeVisible();
    expect(screen.getByLabelText("Challenge")).toBeVisible();
  });

  it("does not allow NDES option to be selected if there is already an NDES CA added", async () => {
    const customRender = createCustomRenderer({
      context: {
        app: {
          config: createMockConfig({
            integrations: {
              zendesk: [],
              jira: [],
              ndes_scep_proxy: {
                url: "https://ndes.example.com",
                admin_url: "https://ndes.example.com/admin",
                username: "ndes_user",
                password: "ndes_password",
              },
            },
          }),
        },
      },
    });

    const { user } = customRender(<AddCertAuthorityModal onExit={noop} />);

    // testing library does not see options when it is disabled
    // so we can just check that its not queryable to confirm that it is disabled
    await user.click(screen.getByRole("combobox"));
    expect(
      screen.queryByRole("option", {
        name: "Microsoft NDES (Network Device Enrollment Service)",
      })
    ).toBeNull();
  });
});
