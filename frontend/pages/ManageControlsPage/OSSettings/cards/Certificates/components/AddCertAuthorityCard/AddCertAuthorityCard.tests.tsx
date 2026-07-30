import React from "react";

import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import PATHS from "router/paths";

import AddCertAuthorityCard from "./AddCertAuthorityCard";

describe("AddCertAuthorityCard", () => {
  it("renders the empty-state copy and Add CA button", () => {
    const router = createMockRouter();
    const render = createCustomRenderer();
    render(<AddCertAuthorityCard router={router} />);

    expect(screen.getByText("Add certificate authority")).toBeInTheDocument();
    expect(
      screen.getByText(/custom SCEP certificate authority must be configured/i)
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add CA" })).toBeInTheDocument();
  });

  it("routes to the certificate authorities settings page when Add CA is clicked", async () => {
    const router = createMockRouter();
    const render = createCustomRenderer();
    const { user } = render(<AddCertAuthorityCard router={router} />);

    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(router.push).toHaveBeenCalledWith(
      PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES
    );
  });
});
