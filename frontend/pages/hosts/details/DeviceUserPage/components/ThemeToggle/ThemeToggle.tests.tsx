import React from "react";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ThemeToggle from "./ThemeToggle";

const THEME_KEY = "fleet-theme";

describe("ThemeToggle", () => {
  beforeEach(() => {
    localStorage.removeItem(THEME_KEY);
    document.body.classList.remove("dark-mode");
  });

  it("renders three theme options", () => {
    render(<ThemeToggle />);
    expect(
      screen.getByRole("radio", { name: "Light mode" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("radio", { name: "Sync with system" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("radio", { name: "Dark mode" })
    ).toBeInTheDocument();
  });

  it("marks the stored mode as checked", () => {
    localStorage.setItem(THEME_KEY, "dark");
    render(<ThemeToggle />);
    expect(screen.getByRole("radio", { name: "Dark mode" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByRole("radio", { name: "Light mode" })).toHaveAttribute(
      "aria-checked",
      "false"
    );
  });

  it("defaults to system when nothing is stored", () => {
    render(<ThemeToggle />);
    expect(
      screen.getByRole("radio", { name: "Sync with system" })
    ).toHaveAttribute("aria-checked", "true");
  });

  it("persists the selected mode and updates active state on click", async () => {
    const user = userEvent.setup();
    render(<ThemeToggle />);

    await user.click(screen.getByRole("radio", { name: "Dark mode" }));

    expect(localStorage.getItem(THEME_KEY)).toBe("dark");
    expect(screen.getByRole("radio", { name: "Dark mode" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });

  it("clears storage when selecting system", async () => {
    const user = userEvent.setup();
    localStorage.setItem(THEME_KEY, "dark");
    render(<ThemeToggle />);

    await user.click(screen.getByRole("radio", { name: "Sync with system" }));

    expect(localStorage.getItem(THEME_KEY)).toBeNull();
  });

  it("stays in sync when theme changes elsewhere", () => {
    render(<ThemeToggle />);

    act(() => {
      localStorage.setItem(THEME_KEY, "light");
      window.dispatchEvent(
        new CustomEvent("fleet-theme-change", { detail: { dark: false } })
      );
    });

    expect(screen.getByRole("radio", { name: "Light mode" })).toHaveAttribute(
      "aria-checked",
      "true"
    );
  });
});
