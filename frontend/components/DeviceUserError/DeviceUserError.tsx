import React from "react";

import ErrorIcon from "../../../assets/images/icon-error-16x16@2x.png";

const baseClass = "device-user-error";

const DeviceUserError = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className="info">
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            This URL is invalid or expired.
          </span>
          <span className="info__data">
            To access your device information, please click “My Device” from the
            Fleet Desktop menu icon.
          </span>
        </div>
      </div>
    </div>
  );
};

export default DeviceUserError;
