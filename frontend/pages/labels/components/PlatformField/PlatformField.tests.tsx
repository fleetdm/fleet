import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import PlatformField from "./PlatformField";

describe("PlatformField component", () => {
  it("renders all platform options, without distro-specific Linux options", async () => {
    const onChange = jest.fn();
    render(<PlatformField platform="" onChange={onChange} />);

    // Open the dropdown.
    await userEvent.click(screen.getByText(/all platforms/i));

    expect(screen.getByText("macOS")).toBeInTheDocument();
    expect(screen.getByText("Windows")).toBeInTheDocument();
    expect(screen.getByText("Linux")).toBeInTheDocument();
    expect(screen.queryByText(/ubuntu/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/centos/i)).not.toBeInTheDocument();
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

});
