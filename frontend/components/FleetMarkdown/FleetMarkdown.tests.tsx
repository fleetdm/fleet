import React from "react";
import { render, screen } from "@testing-library/react";
import FleetMarkdown from "./FleetMarkdown";

jest.mock("components/SQLEditor", () => ({
  __esModule: true,
  default: ({ value }: { value: string }) => (
    <pre data-testid="sql-editor">{value}</pre>
  ),
}));

/**
 * Tests Fleet-specific rendering behavior in the FleetMarkdown wrapper:
 * CustomLink integration, SQLEditor code block delegation, and inline
 * code passthrough.
 */
describe("FleetMarkdown", () => {
  it("renders plain text", () => {
    render(<FleetMarkdown markdown="Hello world" />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("renders links via CustomLink (opens in new tab)", () => {
    render(<FleetMarkdown markdown="Visit [Fleet](https://fleetdm.com)" />);
    const link = screen.getByRole("link", { name: /Fleet/i });
    expect(link).toHaveAttribute("href", "https://fleetdm.com");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders code blocks through SQLEditor", () => {
    const md = "```\nSELECT * FROM users\n```";
    render(<FleetMarkdown markdown={md} />);
    expect(screen.getByTestId("sql-editor")).toHaveTextContent(
      "SELECT * FROM users"
    );
  });

  it("renders inline code without SQLEditor", () => {
    const { container } = render(
      <FleetMarkdown markdown="Run `fleetctl apply`" />
    );
    const code = container.querySelector("code");
    expect(code?.textContent).toBe("fleetctl apply");
    expect(screen.queryByTestId("sql-editor")).not.toBeInTheDocument();
  });
});
