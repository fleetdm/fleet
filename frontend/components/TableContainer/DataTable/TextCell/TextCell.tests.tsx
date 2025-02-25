import React from "react";
import { render, screen } from "@testing-library/react";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TextCell from "./TextCell";

describe("TextCell", () => {
  it("renders booleans as string", () => {
    render(<TextCell value={false} />);
    expect(screen.getByText("false")).toBeInTheDocument();
  });

  it("renders a default value when `value` is null, undefined, or an empty string", () => {
    const { rerender } = render(<TextCell value={null} />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
    rerender(<TextCell value={undefined} />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
    rerender(<TextCell value="" />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders a default value when `value` is null, undefined, or an empty string after formatting", () => {
    const { rerender } = render(
      <TextCell value="foo" formatter={() => null} />
    );
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
    rerender(<TextCell value="foo" formatter={() => undefined} />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
    rerender(<TextCell value="foo" formatter={() => ""} />);
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("uses the provided formatter function", () => {
    render(<TextCell value="foo" formatter={() => "bar"} />);
    expect(screen.getByText("bar")).toBeInTheDocument();
  });

  it("renders the value '0' as a number", () => {
    render(<TextCell value={0} />);
    expect(screen.getByText("0")).toBeInTheDocument();
  });
});
