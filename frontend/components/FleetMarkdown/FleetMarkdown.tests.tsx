import React from "react";
import { render, screen } from "@testing-library/react";

// Mock SQLEditor since Ace doesn't run cleanly in jsdom
jest.mock("components/SQLEditor", () => ({
  __esModule: true,
  default: ({ value }: { value: string }) => (
    <pre data-testid="sql-editor">{value}</pre>
  ),
}));

import FleetMarkdown from "./FleetMarkdown";

/**
 * Tests that react-markdown + remark-gfm render markdown content correctly
 * through the FleetMarkdown wrapper component.
 */
describe("FleetMarkdown - react-markdown rendering", () => {
  it("renders plain text", () => {
    render(<FleetMarkdown markdown="Hello world" />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("renders bold and italic text", () => {
    const { container } = render(
      <FleetMarkdown markdown="This is **bold** and *italic* text" />
    );
    expect(container.querySelector("strong")?.textContent).toBe("bold");
    expect(container.querySelector("em")?.textContent).toBe("italic");
  });

  it("renders links via CustomLink (opens in new tab)", () => {
    render(<FleetMarkdown markdown="Visit [Fleet](https://fleetdm.com)" />);
    const link = screen.getByRole("link", { name: /Fleet/i });
    expect(link).toHaveAttribute("href", "https://fleetdm.com");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("renders unordered lists", () => {
    const md = "- Item one\n- Item two\n- Item three";
    const { container } = render(<FleetMarkdown markdown={md} />);
    const items = container.querySelectorAll("li");
    expect(items).toHaveLength(3);
    expect(items[0].textContent).toBe("Item one");
  });

  it("renders GFM tables (remark-gfm)", () => {
    const md = "| Name | Age |\n|------|-----|\n| Alice | 30 |";
    const { container } = render(<FleetMarkdown markdown={md} />);
    expect(container.querySelector("table")).toBeInTheDocument();
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("renders GFM strikethrough (remark-gfm)", () => {
    const { container } = render(
      <FleetMarkdown markdown="This is ~~deleted~~ text" />
    );
    expect(container.querySelector("del")?.textContent).toBe("deleted");
  });

  it("renders code blocks through SQLEditor mock", () => {
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
  });

  it("applies custom className", () => {
    const { container } = render(
      <FleetMarkdown markdown="test" className="my-custom-class" />
    );
    expect(container.firstChild).toHaveClass("fleet-markdown");
    expect(container.firstChild).toHaveClass("my-custom-class");
  });
});
