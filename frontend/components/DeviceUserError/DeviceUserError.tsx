import React from "react";

import classNames from "classnames";
import Icon from "components/Icon/Icon";
import DataError from "components/DataError";

const baseClass = "device-user-error";

interface IDeviceUserErrorProps {
  /** Modifies styling for mobile width (<768px) */
  isMobileView?: boolean;
  /** Modifies error message for iPhone/iPad/Android */
  isMobileDevice?: boolean;
  isAuthenticationError?: boolean;
  isErrorSetupSteps?: boolean;
}

const DeviceUserError = ({
  isMobileView = false,
  isMobileDevice = false,
  isAuthenticationError = false,
  isErrorSetupSteps = false,
}: IDeviceUserErrorProps): JSX.Element => {
  const wrapperClassnames = classNames(baseClass, {
    [`${baseClass}__mobile-view`]: isMobileView,
  });

  // Default: "Something went wrong"
  let headerContent: React.ReactNode = (
    <>
      <Icon name="error" /> Something went wrong
    </>
  );

  let descriptionContent: React.ReactNode = <>Please contact your IT admin.</>;

  if (isErrorSetupSteps) {
    // Use generic UI error component
    return (
      <div className={wrapperClassnames}>
        <div className={`${baseClass}__inner`}>
          <DataError description="Could not get software setup status." />
        </div>
      </div>
    );
  }

  if (isAuthenticationError) {
    headerContent = (
      <>
        <Icon name="error" />
        {isMobileDevice
          ? "Invalid or missing certificate"
          : "This URL is invalid or expired."}
      </>
    );
    descriptionContent = isMobileDevice ? (
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
        <div className={`${baseClass}__content`}>
          <span className={`${baseClass}__header`}>{headerContent}</span>
          <span className={`${baseClass}__description`}>
            {descriptionContent}
          </span>
        </div>
      </div>
    </div>
  );
};

export default DeviceUserError;
