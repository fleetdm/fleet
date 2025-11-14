import React from "react";
import Icon from "components/Icon/Icon";
import classNames from "classnames";

const baseClass = "device-user-error";

interface IDeviceUserErrorProps {
  isMobileView?: boolean;
  isAuthenticationError?: boolean;
  platform?: string;
}

const DeviceUserError = ({
  isMobileView = false,
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
        {isMobileView
          ? "Invalid or missing certificate"
          : "This URL is invalid or expired."}
      </>
    );

    // iOS/iPadOS specific error messaging
    if (isIOSIPadOS) {
      bodyContent = (
        <>
          Couldn't authenticate this device. This may be because:
          <ul style={{ marginTop: "8px", paddingLeft: "20px", textAlign: "left" }}>
            <li>The Fleet Identity certificate is not installed</li>
            <li>The certificate has expired or been revoked</li>
            <li>This device is not enrolled in MDM</li>
          </ul>
          <div style={{ marginTop: "12px" }}>
            Please contact your IT admin for assistance.
          </div>
        </>
      );
    } else {
      bodyContent = isMobileView ? (
        "Couldn't authenticate this device. Please contact your IT admin."
      ) : (
        <>
          To access your device information, please click <br />
          "My Device" from the Fleet Desktop menu icon.
        </>
      );
    }
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
