import React from "react";

import CustomLink from "components/CustomLink";

import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

import MacOSUpdateScreenshot from "../../../../../../assets/images/macos-updates-preview.png";
import WindowsUpdateScreenshot from "../../../../../../assets/images/windows-nudge-screenshot.png";
import IOSUpdateScreenshot from "../../../../../../assets/images/ios-updates-preview.png";
import IPadOSUpdateScreenshot from "../../../../../../assets/images/ipados-updates-preview.png";

const baseClass = "os-requirement-preview";

interface IEndUserOSRequirementPreviewProps {
  platform: OSUpdatesSupportedPlatform;
}
const OSRequirementDescription = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  switch (platform) {
    case "darwin":
      return (
        <>
          <h3>End user experience on macOS</h3>
          <p>
            For macOS 14 and above, end users will see native macOS
            notifications (DDM).
          </p>
          <p>Everyone else will see the Nudge window.</p>
          <CustomLink
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/os-updates"
            newTab
          />
        </>
      );
    case "windows":
      return (
        <>
          <h3>End user experience on Windows</h3>
          <p>
            When a Windows host becomes aware of a new update, end users are
            able to defer restarts. Automatic restarts happen before 8am and
            after 5pm (end user&apos;s local time). After the deadline, restarts
            are forced regardless of active hours.
          </p>
          <CustomLink
            text="Learn more about Windows updates in Fleet"
            url="https://fleetdm.com/learn-more-about/os-updates"
            newTab
          />
        </>
      );
    case "ios":
      return (
        <>
          <h3>End user experience on iOS</h3>
        </>
      );
    case "ipados":
      return (
        <>
          <h3>End user experience on iPadOS</h3>
        </>
      );
    default:
      return <></>;
  }
};

const OSRequirementImage = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  const getScreenshot = () => {
    switch (platform) {
      case "darwin":
        return MacOSUpdateScreenshot;
      case "windows":
        return WindowsUpdateScreenshot;
      case "ios":
        return IOSUpdateScreenshot;
      case "ipados":
        return IPadOSUpdateScreenshot;
      default:
        MacOSUpdateScreenshot;
    }
  };

  return (
    <img
      className={`${baseClass}__preview-img`}
      src={getScreenshot()}
      alt="OS update preview screenshot"
    />
  );
};

const EndUserOSRequirementPreview = ({
  platform,
}: IEndUserOSRequirementPreviewProps) => {
  // FIXME: on slow connection the image loads after the text which looks weird and can cause a
  // mismatch between the text and the image when switching between platforms. We should load the
  // image first and then the text.
  return (
    <div className={baseClass}>
      <OSRequirementDescription platform={platform} />
      <OSRequirementImage platform={platform} />
    </div>
  );
};

export default EndUserOSRequirementPreview;
