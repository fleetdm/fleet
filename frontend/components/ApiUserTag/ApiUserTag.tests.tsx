import { screen, waitFor } from "@testing-library/react";
import React from "react";

import { renderWithSetup } from "test/test-utils";

import ApiUserTag from "./ApiUserTag";

describe("ApiUserTag", () => {
  it("explains on hover that the user only has API access", async () => {
    const { user } = renderWithSetup(<ApiUserTag />);

    await user.hover(screen.getByText("API"));

    await waitFor(() => {
      expect(
        screen.getByText("This user only has API access.")
      ).toBeInTheDocument();
    });
  });
});
