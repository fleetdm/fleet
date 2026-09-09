import React from "react";
import { render, screen } from "@testing-library/react";

import SSOError from "./SSOError";

describe("SSOError", () => {
  it("tells the user their sign-in timed out when the session expired", () => {
    render(<SSOError sessionExpired />);

    expect(
      screen.getByText(/Your session may have timed out/)
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/If this keeps happening/)
    ).not.toBeInTheDocument();
  });

  it("keeps the generic message for every other failure", () => {
    render(<SSOError />);

    expect(screen.getByText(/If this keeps happening/)).toBeInTheDocument();
    expect(screen.queryByText(/timed out/)).not.toBeInTheDocument();
  });
});
