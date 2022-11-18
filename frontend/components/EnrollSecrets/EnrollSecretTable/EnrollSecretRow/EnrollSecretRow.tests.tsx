import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup } from "test/testingUtils";

import EnrollSecretRow from "./EnrollSecretRow";

const TEAM_SECRET = {
  secret: "super-secret-secret",
  created_at: "",
  team_id: 2,
};
describe("Enroll secret row", () => {
  it("Hides secret by default and shows secret on click of eye icon", async () => {
    const { user, debug, container } = renderWithSetup(
      <EnrollSecretRow secret={TEAM_SECRET} />
    );

    // TODO: Figure out how to grab an input, 30 minutes of google no success
    // const secretHidden = screen.getByTestId("osquery-secret");
    // const ok = screen.getByRole("form", { name: /osquery/i });
    // const eyeIcon = screen.findByTestId("eye-icon");
    // debug();
    // const inputEl = container.querySelector(`input[name="osquery-secret-2"]`);
    // expect(inputEl).toHaveAttribute("type", "password");
    // console.log("\n\n\n\n ok", ok);
    // await user.click(await eyeIcon);

    // const secretShown = screen.getByTestId("osquery-secret");

    // expect(secretShown).toHaveAttribute("type", "text");
  });
});
