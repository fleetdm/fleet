import React from "react";

import { render, screen } from "@testing-library/react";

import TeamHostExpiryToggle from "./TeamHostExpiryToggle";

describe("TeamHostExpiryToggle component", () => {
  // global setting disabled
  it("Renders correctly with no global window set", () => {
    render(
      <TeamHostExpiryToggle
        globalHostExpiryEnabled={false}
        globalHostExpiryWindow={undefined}
        teamExpiryEnabled={false}
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable host expiry/)).toBeInTheDocument();
    expect(screen.queryByText(/Host expiry is globally enabled/)).toBeNull();
  });

  // global setting enabled
  it("Renders as expected when global enabled, local disabled", () => {
    render(
      <TeamHostExpiryToggle
        globalHostExpiryEnabled
        globalHostExpiryWindow={2}
        teamExpiryEnabled={false}
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable host expiry/)).toBeInTheDocument();
    expect(
      screen.getByText(/Host expiry is globally enabled/)
    ).toBeInTheDocument();
    expect(screen.getByText(/Add custom expiry window/)).toBeInTheDocument();
  });

  it("Renders as expected when global enabled, local enabled", () => {
    render(
      <TeamHostExpiryToggle
        globalHostExpiryEnabled
        globalHostExpiryWindow={2}
        teamExpiryEnabled
        setTeamExpiryEnabled={jest.fn()}
      />
    );

    expect(screen.getByText(/Enable host expiry/)).toBeInTheDocument();
    expect(
      screen.getByText(/Host expiry is globally enabled/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/Add custom expiry window/)).toBeNull();
  });
});
