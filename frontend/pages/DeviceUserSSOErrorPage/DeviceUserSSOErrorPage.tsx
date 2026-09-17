import React from "react";
import { WithRouterProps } from "react-router";

import DeviceUserError from "components/DeviceUserError";

const baseClass = "device-user-sso-error-page";

interface IDeviceUserSSOErrorQuery {
  reason?: string;
}

const DeviceUserSSOErrorPage = ({
  location: { query },
}: WithRouterProps<object, IDeviceUserSSOErrorQuery>) => {
  return (
    <div className={`${baseClass} app-wrap`}>
      <DeviceUserError
        ssoError={
          query.reason === "session_expired"
            ? "session_expired"
            : "callback_failed"
        }
      />
    </div>
  );
};

export default DeviceUserSSOErrorPage;
