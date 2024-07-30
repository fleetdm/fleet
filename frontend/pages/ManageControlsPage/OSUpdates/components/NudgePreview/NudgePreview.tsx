import React from "react";

import CustomLink from "components/CustomLink";

import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

import MacOSUpdateScreenshot from "../../../../../../assets/images/macos-updates-preview.png";
import WindowsUpdateScreenshot from "../../../../../../assets/images/windows-nudge-screenshot.png";

const baseClass = "nudge-preview";

interface INudgeDescriptionProps {
  platform: OSUpdatesSupportedPlatform;
}
const NudgeDescription = ({ platform }: INudgeDescriptionProps) => {
  return platform === "darwin" ? (
    <>
      <h3>End user experience on macOS</h3>
      <p>
        For macOS 14 and above, end users will see native macOS notifications
        (DDM).
      </p>
      <p>Everyone else will see the Nudge window.</p>
      <CustomLink
        text="Learn more"
        url="https://fleetdm.com/learn-more-about/os-updates"
        newTab
      />
    </>
  ) : (
    <>
      <h3>End user experience on Windows</h3>
      <p>
        When a Windows host becomes aware of a new update, end users are able to
        defer restarts. Automatic restarts happen before 8am and after 5pm (end
        user&apos;s local time). After the deadline, restarts are forced
        regardless of active hours.
      </p>
      <CustomLink
        text="Learn more about Windows updates in Fleet"
        url="https://fleetdm.com/learn-more-about/os-updates"
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
      src={
        platform === "darwin" ? MacOSUpdateScreenshot : WindowsUpdateScreenshot
      }
      alt="OS update preview screenshot"
    />
  );
};

interface INudgePreviewProps {
  platform: OSUpdatesSupportedPlatform;
}

const NudgePreview = ({ platform }: INudgePreviewProps) => {
  const isSupportedPlatform = platform === "windows" || platform === "darwin";

  if (!isSupportedPlatform) {
    return null;
  }

  // FIXME: on slow connection the image loads after the text which looks weird and can cause a
  // mismatch between the text and the image when switching between platforms. We should load the
  // image first and then the text.
  return (
    <div className={baseClass}>
      <NudgeDescription platform={platform} />
      <NudgeImage platform={platform} />
    </div>
  );
};

export default NudgePreview;
