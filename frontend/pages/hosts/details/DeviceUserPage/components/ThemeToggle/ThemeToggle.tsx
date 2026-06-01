import React, { useEffect, useState } from "react";
import classnames from "classnames";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import { getThemeMode, setThemeMode, ThemeMode } from "utilities/theme";

const baseClass = "theme-toggle";

interface IThemeOption {
  mode: ThemeMode;
  icon: IconNames;
  label: string;
}

const OPTIONS: IThemeOption[] = [
  { mode: "light", icon: "sun", label: "Light mode" },
  { mode: "system", icon: "theme-auto", label: "Sync with system" },
  { mode: "dark", icon: "moon", label: "Dark mode" },
];

const ThemeToggle = () => {
  const [mode, setMode] = useState<ThemeMode>(getThemeMode);

  // Stay in sync if the theme is changed elsewhere (e.g. another tab via the
  // storage event, or a future second toggle on the page).
  useEffect(() => {
    const handler = () => setMode(getThemeMode());
    window.addEventListener("fleet-theme-change", handler);
    return () => window.removeEventListener("fleet-theme-change", handler);
  }, []);

  const onSelect = (next: ThemeMode) => {
    if (next === mode) return;
    setThemeMode(next);
    setMode(next);
  };

  return (
    <div className={baseClass} role="radiogroup" aria-label="Color theme">
      {OPTIONS.map(({ mode: optionMode, icon, label }) => {
        const isActive = mode === optionMode;
        return (
          <button
            key={optionMode}
            type="button"
            role="radio"
            aria-checked={isActive}
            aria-label={label}
            title={label}
            tabIndex={isActive ? 0 : -1}
            className={classnames(`${baseClass}__option`, {
              [`${baseClass}__option--active`]: isActive,
            })}
            onClick={() => onSelect(optionMode)}
          >
            <Icon
              name={icon}
              size="small"
              color={isActive ? "core-fleet-black" : "ui-fleet-black-50"}
            />
          </button>
        );
      })}
    </div>
  );
};

export default ThemeToggle;
