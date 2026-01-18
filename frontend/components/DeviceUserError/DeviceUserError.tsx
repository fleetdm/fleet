import React from "react";
import classNames from "classnames";
import Icon from "components/Icon/Icon";

const baseClass = "device-user-error";

interface IDeviceUserErrorProps {
  /** Modifies styling for mobile width (<768px) */
  isMobileView?: boolean;
  /** Modifies error message for iPhone/iPad/Android */
  isMobileDevice?: boolean;
  isAuthenticationError?: boolean;
  platform?: string;
}

const DeviceUserError = ({
  isMobileView = false,
  isMobileDevice = false,
  isAuthenticationError = false,
  platform,
}: IDeviceUserErrorProps): JSX.Element => {
  const wrapperClassnames = classNames(baseClass, {
    [`${baseClass}__mobile-view`]: isMobileView,
  });

  const isIOSIPadOS = platform === "ios" || platform === "ipados";

  // Default: "Something went wrong"
  let headerContent: React.ReactNode = (
    <>
      <Icon name="error" /> Something went wrong
    </>
  );
  let bodyContent: React.ReactNode = <>Please contact your IT admin.</>;

  if (isAuthenticationError) {
    headerContent = (
      <>
        <Icon name="error" />
        {isMobileDevice
          ? "Invalid or missing certificate"
          : "This URL is invalid or expired."}
      </>
    );
    bodyContent = isMobileDevice ? (
      "Couldn't authenticate this device. Please contact your IT admin."
    ) : (
      <>
        To access your device information, please click <br />
        “My Device” from the Fleet Desktop menu icon.
      </>
    );
  }

  return (
    <div className={wrapperClassnames}>
      <div className={`${baseClass}__inner`}>
        <div className="info">
          <span className="info__header">{headerContent}</span>
          <span className="info__data">{bodyContent}</span>
        </div>
      </div>
    </div>
  );
};

export default DeviceUserError;
