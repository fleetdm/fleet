import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import PlatformField from "./PlatformField";

describe("PlatformField component", () => {
  it("renders all platform options, including generic Linux", async () => {
    const onChange = jest.fn();
    render(<PlatformField platform="" onChange={onChange} />);

    // Open the dropdown.
    await userEvent.click(screen.getByText(/all platforms/i));

    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Windows")).toBeInTheDocument();
    expect(screen.getByText("Linux")).toBeInTheDocument();
    expect(screen.getByText("Ubuntu (Linux)")).toBeInTheDocument();
    expect(screen.getByText("CentOS (Linux)")).toBeInTheDocument();
  });

  it("calls onChange with the platform value when Linux is selected", async () => {
    const onChange = jest.fn();
    render(<PlatformField platform="" onChange={onChange} />);

    await userEvent.click(screen.getByText(/all platforms/i));
    await userEvent.click(screen.getByText("Linux"));

    expect(onChange).toHaveBeenCalledWith("linux");
  });

  it("renders read-only display text when editing", () => {
    render(<PlatformField platform="linux" isEditing />);

    expect(screen.getByText("Linux")).toBeInTheDocument();
  });

  it("renders updated display text for ubuntu and centos when editing", () => {
    const { rerender } = render(<PlatformField platform="ubuntu" isEditing />);
    expect(screen.getByText("Ubuntu (Linux)")).toBeInTheDocument();

    rerender(<PlatformField platform="centos" isEditing />);
    expect(screen.getByText("CentOS (Linux)")).toBeInTheDocument();
  });
});
