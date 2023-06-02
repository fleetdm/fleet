import React from "react";

import Icon from "components/Icon/Icon";

const baseClass = "device-user-error";

const DeviceUserError = (): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <div className={`${baseClass}__inner`}>
        <div className="info">
          <span className="info__header">
            <Icon name="error-outline" />
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
