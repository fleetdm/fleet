import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import CopyButton from "./CopyButton";

describe("CopyButton component", () => {
  beforeEach(() => {
    Object.assign(navigator, {
      clipboard: { writeText: jest.fn().mockResolvedValue(undefined) },
    });
  });

  it("renders the copy icon by default", () => {
    render(<CopyButton copyText="abc" />);
    expect(screen.getByTestId("copy-icon")).toBeInTheDocument();
  });

  it("copies text to the clipboard on click", async () => {
    const writeText = jest
      .spyOn(navigator.clipboard, "writeText")
      .mockResolvedValue(undefined);
    render(<CopyButton copyText="hello" />);
    await userEvent.click(screen.getByRole("button"));
    expect(writeText).toHaveBeenCalledWith("hello");
  });

  it('shows "Copied!" after a successful copy', async () => {
    render(<CopyButton copyText="hello" />);
    await userEvent.click(screen.getByRole("button"));
    await waitFor(() =>
      expect(screen.getByText("Copied!")).toBeInTheDocument()
    );
  });

  it('shows "Copy failed" if the clipboard call rejects', async () => {
    jest
      .spyOn(navigator.clipboard, "writeText")
      .mockRejectedValueOnce(new Error("blocked"));
    render(<CopyButton copyText="hello" />);
    await userEvent.click(screen.getByRole("button"));
    await waitFor(() =>
      expect(screen.getByText("Copy failed")).toBeInTheDocument()
    );
  });

  it("renders custom children when provided", () => {
    render(
      <CopyButton copyText="abc">
        <span>Copy me</span>
      </CopyButton>
    );
    expect(screen.getByText("Copy me")).toBeInTheDocument();
  });
});
