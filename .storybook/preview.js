import React, { useEffect } from "react";
// Mirrors frontend/index.jsx: react-select v1 base CSS must load before
// index.scss so Fleet's `.Select-*` overrides (e.g. dark-mode label color) win.
import "react-select/dist/react-select.css";
import "../frontend/index.scss";
import "./preview.scss";

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
