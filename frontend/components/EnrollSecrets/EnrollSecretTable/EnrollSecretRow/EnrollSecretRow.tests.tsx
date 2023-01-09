import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import EnrollSecretRow from "./EnrollSecretRow";

const TEAM_SECRET = {
  secret: "super-secret-secret",
  created_at: "",
  team_id: 2,
};
describe("Enroll secret row", () => {
  it("Hides secret by default and shows secret on click of eye icon", async () => {
    const { user, container } = renderWithSetup(
      <EnrollSecretRow secret={TEAM_SECRET} />
    );

    // Secret hidden by default
    const secretHidden = container.querySelector("input");
    expect(secretHidden?.type === "password").toBeTruthy();

    // Click eye icon
    const eyeIcon = screen.getByTestId("eye-icon");
    await user.click(eyeIcon);

    // Secret shown
    const secretShown = container.querySelector("input");
    expect(secretShown?.type === "text").toBeTruthy();
  });
});
