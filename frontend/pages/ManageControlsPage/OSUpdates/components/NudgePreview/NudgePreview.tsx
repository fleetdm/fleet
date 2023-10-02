import React from "react";

import CustomLink from "components/CustomLink";

import OsUpdateScreenshot from "../../../../../../assets/images/nudge-screenshot.png";

const baseClass = "nudge-preview";

const NudgePreview = () => {
  return (
    <div className={baseClass}>
      <h2>End user experience</h2>
      <p>
        When a minimum version is saved, the end user sees the below window
        until their macOS version is at or above the minimum version.
      </p>
      <p>As the deadline gets closer, Fleet provides stronger encouragement.</p>
      <CustomLink
        text="Learn more about macOS updates in Fleet"
        url="https://fleetdm.com/docs/using-fleet/mdm-macos-updates"
        newTab
      />
      <img
        className={`${baseClass}__preview-img`}
        src={OsUpdateScreenshot}
        alt="OS update preview screenshot"
      />
    </div>
  );
};

export default NudgePreview;
