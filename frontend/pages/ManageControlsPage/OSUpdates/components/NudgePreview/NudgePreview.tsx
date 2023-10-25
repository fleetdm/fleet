import React from "react";

import CustomLink from "components/CustomLink";

import MacOSUpdateScreenshot from "../../../../../../assets/images/nudge-screenshot.png";

const baseClass = "nudge-preview";

interface INudgeDescriptionProps {
  platform: "mac" | "windows";
}
const NudeDescription = ({ platform }: INudgeDescriptionProps) => {
  return platform === "mac" ? (
    <>
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
    </>
  ) : (
    <>
      <p>
        When a new Windows update is published, the update will be downloaded
        and installed automatically before 8am and after 5pm (end user’s local
        time). Before the deadline passes, users will be able to defer restarts.
        After the deadline passes restart will be forced regardless of active
        hours.
      </p>
      <CustomLink
        text="Learn more about Windows updates in Fleet"
        url="Links to: https://fleetdm.com/docs/using-fleet/mdm-windows-updates"
        newTab
      />
    </>
  );
};

type INudgeImageProps = INudgeDescriptionProps;

const NudgeImage = ({ platform }: INudgeImageProps) => {
  return (
    <img
      className={`${baseClass}__preview-img`}
      src={platform === "mac" ? MacOSUpdateScreenshot : MacOSUpdateScreenshot}
      alt="OS update preview screenshot"
    />
  );
};

interface INudgePreviewProps {
  platform: "mac" | "windows";
}

const NudgePreview = ({ platform }: INudgePreviewProps) => {
  return (
    <div className={baseClass}>
      <h2>End user experience</h2>
      <NudeDescription platform={platform} />
      <NudgeImage platform={platform} />
    </div>
  );
};

export default NudgePreview;
