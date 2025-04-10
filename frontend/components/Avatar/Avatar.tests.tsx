import React from "react";
import { render, screen } from "@testing-library/react";

import Avatar from "./Avatar";

describe("Avatar - component", () => {
  it("renders the user gravatar if provided", () => {
    render(
      <Avatar user={{ gravatar_url: "https://example.com/avatar.png" }} />
    );

    const avatar = screen.getByAltText("User avatar");
    expect(avatar).toBeVisible();
    expect(avatar).toHaveAttribute("src", "https://example.com/avatar.png");
  });

  it("renders the fleet avatar if useFleetAvatar is `true`", () => {
    render(<Avatar useFleetAvatar user={{}} />);
    expect(screen.getByTestId("fleet-avatar")).toBeVisible();
  });
});
