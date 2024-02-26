import React from "react";
import { render, screen } from "@testing-library/react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TextCell from "./TextCell";

describe("TextCell", () => {
  it("renders booleans as string", () => {
    render(<TextCell value={false} />);
    expect(screen.getByText("false")).toBeInTheDocument();
  });

  it("renders a default value when `value` is empty", () => {
    render(<TextCell value="" />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders a default value when `value` is empty after formatting", () => {
    render(<TextCell value="foo" formatter={() => ""} />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("uses the provided formatter function", () => {
    render(<TextCell value="foo" formatter={() => "bar"} />);
    expect(screen.getByText("bar")).toBeInTheDocument();
  });
});
