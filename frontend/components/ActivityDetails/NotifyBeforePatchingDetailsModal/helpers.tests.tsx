import { render, screen } from "@testing-library/react";
import React from "react";

import { renderNotifyTitleList } from "./helpers";

describe("renderNotifyTitleList", () => {
  it("returns null for an empty list", () => {
    expect(renderNotifyTitleList([])).toBeNull();
  });

  it("renders a single legacy string title", () => {
    render(<>{renderNotifyTitleList(["1Password"])}</>);
    expect(screen.getByText("1Password")).toBeInTheDocument();
  });

  it("prefers display_name when the override is set", () => {
    render(
      <>
        {renderNotifyTitleList([
          { title: "1Password", display_name: "1Password 8", title_id: 1 },
        ])}
      </>
    );
    expect(screen.getByText("1Password 8")).toBeInTheDocument();
    expect(screen.queryByText("1Password")).not.toBeInTheDocument();
  });

  it("falls back to title when display_name is absent", () => {
    render(<>{renderNotifyTitleList([{ title: "Slack", title_id: 2 }])}</>);
    expect(screen.getByText("Slack")).toBeInTheDocument();
  });

  it("joins two titles with 'and', preferring display_name per-item", () => {
    render(
      <>
        {renderNotifyTitleList([
          { title: "1Password", display_name: "1Password 8", title_id: 1 },
          { title: "Slack", title_id: 2 },
        ])}
      </>
    );
    expect(screen.getByText("1Password 8")).toBeInTheDocument();
    expect(screen.getByText("Slack")).toBeInTheDocument();
  });

  it("truncates past four titles with a 'N more apps' suffix", () => {
    render(
      <>
        {renderNotifyTitleList([
          { title: "AppOne", title_id: 1 },
          { title: "AppTwo", title_id: 2 },
          { title: "AppThree", title_id: 3 },
          { title: "AppFour", title_id: 4 },
          { title: "AppFive", title_id: 5 },
        ])}
      </>
    );
    expect(screen.getByText("AppOne")).toBeInTheDocument();
    expect(screen.getByText("AppTwo")).toBeInTheDocument();
    expect(screen.getByText("AppThree")).toBeInTheDocument();
    expect(screen.queryByText("AppFour")).not.toBeInTheDocument();
    expect(screen.getByText(/2 more apps/)).toBeInTheDocument();
  });
});
