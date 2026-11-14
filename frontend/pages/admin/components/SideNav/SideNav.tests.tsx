import { render, screen } from "@testing-library/react";
import React from "react";

import SideNav, { ISideNavItem } from "./SideNav";

const Card = () => <div>Card content</div>;

const navItems: ISideNavItem<unknown>[] = [
  { title: "Visible", urlSection: "visible", path: "/visible", Card },
  {
    title: "Excluded",
    urlSection: "excluded",
    path: "/excluded",
    Card,
    exclude: true,
  },
];

describe("SideNav", () => {
  it("hides excluded nav items but still renders the current card", () => {
    render(
      <SideNav
        navItems={navItems}
        activeItem="excluded"
        CurrentCard={<Card />}
      />
    );

    expect(screen.getByText("Visible")).toBeInTheDocument();
    expect(screen.queryByText("Excluded")).not.toBeInTheDocument();
    expect(screen.getByText("Card content")).toBeInTheDocument();
  });
});
