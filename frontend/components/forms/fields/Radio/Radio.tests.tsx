import React from "react";
import { render, screen } from "@testing-library/react";

import Radio from "./Radio";

describe("Radio - component", () => {
  it("renders the correct selected state", () => {
    render(
      <Radio
        checked
        label={"Test Radio"}
        value={"Test Radio"}
        id={"test-radio"}
        onChange={() => {
          return null;
        }}
        name={"Test Radio"}
      />
    );

    const radio = screen.getByRole("radio", { name: "Test Radio" });
    expect(radio).toBeChecked();
  });

  it("renders the correct disabled state", () => {
    render(
      <Radio
        disabled
        label={"Test Radio"}
        value={"Test Radio"}
        id={"test-radio"}
        onChange={() => {
          return null;
        }}
        name={"Test Radio"}
      />
    );

    const radio = screen.getByRole("radio", { name: "Test Radio" });
    expect(radio).toBeDisabled();
  });
});
