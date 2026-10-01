import { render, screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import DeleteAbmModal from "./DeleteAbmModal";

const renderModal = (props: { tokenIsDefault: boolean; tokensCount: number }) =>
  render(
    <DeleteAbmModal
      tokenOrgName="Acme Inc."
      tokenId={1}
      onCancel={noop}
      onDeletedToken={noop}
      {...props}
    />
  );

describe("DeleteAbmModal", () => {
  it("omits default token copy when deleting the only token", () => {
    renderModal({ tokenIsDefault: true, tokensCount: 1 });

    expect(
      screen.getByText(/won't automatically enroll to Fleet/)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/will become the default|set a new default/)
    ).not.toBeInTheDocument();
  });

  it("omits default token copy when deleting a non-default token", () => {
    renderModal({ tokenIsDefault: false, tokensCount: 3 });

    expect(
      screen.getByText(/won't automatically enroll to Fleet/)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/will become the default|set a new default/)
    ).not.toBeInTheDocument();
  });

  it("says the remaining token becomes the default when deleting the default of two", () => {
    renderModal({ tokenIsDefault: true, tokensCount: 2 });

    expect(
      screen.getByText(
        /Your remaining token will become the default automatically/
      )
    ).toBeInTheDocument();
  });

  it("warns about manual enrollments when deleting the default of several", () => {
    renderModal({ tokenIsDefault: true, tokensCount: 3 });

    expect(
      screen.getByText(
        /Manual enrollments may not be able to sign into Managed Apple IDs until you set a new default/
      )
    ).toBeInTheDocument();
  });
});
