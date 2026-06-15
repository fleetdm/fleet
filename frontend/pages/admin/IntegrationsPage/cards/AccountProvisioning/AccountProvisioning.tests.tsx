import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";

import AccountProvisioning from "./AccountProvisioning";

describe("AccountProvisioning", () => {
  const render = createCustomRenderer({});

  it("renders the section heading", () => {
    render(<AccountProvisioning />);
    expect(screen.getByText("Account provisioning")).toBeInTheDocument();
  });
});
