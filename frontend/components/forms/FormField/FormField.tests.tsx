import { render, screen } from "@testing-library/react";
import React from "react";

import CustomLink from "components/CustomLink";

import FormField from "./FormField";

describe("FormField", () => {
  it("renders the small external link icon for links in help text", () => {
    render(
      <FormField
        label="Name"
        name="name"
        helpText={
          <CustomLink url="https://fleetdm.com/docs" text="Learn more" newTab />
        }
      >
        <input id="name" />
      </FormField>
    );

    expect(
      screen.getByTestId("external-link-icon").querySelector("svg")
    ).toHaveAttribute("width", "12");
  });
});
