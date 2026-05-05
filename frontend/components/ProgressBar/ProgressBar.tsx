import React from "react";
import { COLORS } from "styles/var/colors";

import classnames from "classnames";

const baseClass = "progress-bar";

export interface IProgressBarSection {
  color: string;
  portion: number; // Value between 0 and 1
}

export interface IProgressBar {
  sections: IProgressBarSection[];
  backgroundColor?: string;
  width?: "small" | "large";
}

const ProgressBar = ({
  sections,
  backgroundColor = COLORS["ui-fleet-black-10"],
  width = "large",
}: IProgressBar) => {
  const classes = classnames(baseClass, {
    [`${baseClass}__small`]: width === "small",
    [`${baseClass}__large`]: width === "large",
  });
  return (
    <div className={classes} style={{ backgroundColor }} role="progressbar">
      {sections.map((section, index) => (
        <div
          key={`${section.color}-${section.portion}`}
          data-testid={`section-${index}`}
          className={`${baseClass}__section`}
          style={{
            backgroundColor: section.color,
            width: `${section.portion * 100}%`,
          }}
        />
      ))}
    </div>
  );
};

export default ProgressBar;
