import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import DeleteAbmModal from "./DeleteAbmModal";

describe("DeleteAbmModal", () => {
  it("omits default token copy when deleting the only token", () => {
    render(
      <DeleteAbmModal
        tokenOrgName="Acme Inc."
        tokenId={1}
        tokensCount={1}
        onCancel={noop}
        onDeletedToken={noop}
      />
    );

    expect(
      screen.getByText(/won't automatically enroll to Fleet/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/primary/)).not.toBeInTheDocument();
  });

  it("says the remaining token becomes primary when one token will remain", () => {
    render(
      <DeleteAbmModal
        tokenOrgName="Acme Inc."
        tokenId={1}
        tokensCount={2}
        onCancel={noop}
        onDeletedToken={noop}
      />
    );

    expect(
      screen.getByText(/Your remaining token will become primary automatically/)
    ).toBeInTheDocument();
  });

  it("warns about manual enrollments when multiple tokens will remain", () => {
    render(
      <DeleteAbmModal
        tokenOrgName="Acme Inc."
        tokenId={1}
        tokensCount={3}
        onCancel={noop}
        onDeletedToken={noop}
      />
    );

    expect(
      screen.getByText(
        /Manual enrollments may not be able to sign into Managed Apple IDs until you set a new primary/
      )
    ).toBeInTheDocument();
  });
});
