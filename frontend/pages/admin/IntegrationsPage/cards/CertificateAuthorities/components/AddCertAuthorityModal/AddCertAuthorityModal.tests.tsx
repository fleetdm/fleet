import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { createMockCertificateAuthorityPartial } from "__mocks__/certificatesMock";
import { renderWithSetup } from "test/test-utils";

import AddCertAuthorityModal from "./AddCertAuthorityModal";
import CA_LABEL_BY_TYPE from "../helpers";

describe("AddCertAuthorityModal", () => {
  it("renders the Custom EST form by default", () => {
    render(<AddCertAuthorityModal certAuthorities={[]} onExit={noop} />);

    expect(screen.getByText(CA_LABEL_BY_TYPE.custom_est_proxy)).toBeVisible();
    expect(screen.getByLabelText("Name")).toBeVisible();
    expect(screen.getByLabelText("URL")).toBeVisible();
    expect(screen.getByLabelText("Username")).toBeVisible();
    expect(screen.getByLabelText("Password")).toBeVisible();
  });

  it("shows the correct form when the corresponding value in the dropdown is selected.", async () => {
    const { user } = renderWithSetup(
      <AddCertAuthorityModal certAuthorities={[]} onExit={noop} />
    );

    // this is selecting the custom scep option from the dropdown
    await user.click(screen.getByRole("combobox"));
    await user.click(
      screen.getByRole("option", {
        name: CA_LABEL_BY_TYPE.custom_scep_proxy,
      })
    );

    expect(screen.getByLabelText("Name")).toBeVisible();
    expect(screen.getByLabelText("SCEP URL")).toBeVisible();
    expect(screen.getByLabelText("Challenge")).toBeVisible();
  });

  it("does not allow NDES option to be selected if there is already an NDES CA added", async () => {
    const { user } = renderWithSetup(
      <AddCertAuthorityModal
        certAuthorities={[
          createMockCertificateAuthorityPartial({ type: "ndes_scep_proxy" }),
        ]}
        onExit={noop}
      />
    );

    // testing library does not see options when it is disabled
    // so we can just check that its not queryable to confirm that it is disabled
    await user.click(screen.getByRole("combobox"));
    expect(
      screen.queryByRole("option", {
        name: CA_LABEL_BY_TYPE.ndes_scep_proxy,
      })
    ).toBeNull();
  });
});
