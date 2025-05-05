import React from "react";

import Icon from "components/Icon/Icon";

const baseClass = "device-user-error";

interface IDeviceUserErrorProps {
  /** the description text displayed under the header */
  description?: string;
}

const DeviceUserError = ({
  description,
}: IDeviceUserErrorProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__inner`}>
        <div className="info">
          <span className="info__header">
            <Icon name="error-outline" />
            This URL is invalid or expired.
          </span>
          {description && <span className="info__data">{description}</span>}
          <span className="info__data">
            To access your device information, please click <br />
            “My Device” from the Fleet Desktop menu icon.
          </span>
        </div>
      </div>
    </div>
  );
};

export default DeviceUserError;
