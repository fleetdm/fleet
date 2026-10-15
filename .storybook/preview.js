import React, { useEffect } from "react";

// Mirrors frontend/index.jsx: react-select v1 base CSS must load before
// index.scss so Fleet's `.Select-*` overrides (e.g. dark-mode label color) win.
import "react-select/dist/react-select.css";
import "../frontend/index.scss";
import "./preview.scss";

// Freeze Storybook's clock. `dateAgo` and friends use `formatDistanceToNow`,
// which reads `Date.now()` — without this, the relative label ("5 months ago")
// tips over every real-world month boundary and visual-regression snapshots
// drift even for stories that use fixed `created_at` timestamps.
const MOCK_NOW = new Date("2026-06-01T12:00:00.000Z").getTime();
Date.now = () => MOCK_NOW;

export const globalTypes = {
  theme: {
    description: "Toggle dark/light mode",
    defaultValue: "light",
    toolbar: {
      items: [
        { value: "light", icon: "sun", title: "Light mode" },
        { value: "dark", icon: "moon", title: "Dark mode" },
      ],
      dynamicTitle: true,
    },
  },
};

const applyTheme = (isDark) => {
  document.body.classList.toggle("dark-mode", isDark);
  document.body.style.backgroundColor = isDark ? "var(--core-fleet-white)" : "";
  document.body.style.color = isDark ? "var(--core-fleet-black)" : "";

  document.querySelectorAll(".docs-story").forEach((el) => {
    el.style.backgroundColor = isDark ? "var(--core-fleet-white)" : "";
  });
};

const withTheme = (Story, context) => {
  const isDark = context.globals.theme === "dark";

  useEffect(() => {
    applyTheme(isDark);
  }, [isDark]);

  return <Story />;
};

export const decorators = [withTheme];

export const parameters = {
  controls: {
    matchers: {
      color: /(background|color)$/i,
      date: /Date$/,
    },
  },
};

export const tags = ["autodocs"];
