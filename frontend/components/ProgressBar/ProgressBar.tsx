import React from "react";
import { COLORS } from "styles/var/colors";

const baseClass = "progress-bar";

export interface IProgressBarSection {
  color: string;
  portion: number; // Value between 0 and 1
}

export interface IProgressBar {
  sections: IProgressBarSection[];
  backgroundColor?: string;
}

const ProgressBar = ({
  sections,
  backgroundColor = COLORS["ui-fleet-black-10"],
}: IProgressBar) => {
  return (
    <div
      className={`${baseClass}`}
      style={{ backgroundColor }}
      role="progressbar"
    >
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
