import React from "react";
import { WithRouterProps } from "react-router";

import useIsMobileWidth from "hooks/useIsMobileWidth";
import DeviceUserError from "components/DeviceUserError";

const baseClass = "device-user-sso-error-page";

interface IDeviceUserSSOErrorQuery {
  reason?: string;
}

/**
 * Where the SSO callback sends a Fleet Desktop end user when it fails before it
 * can load the SSO session, which is where the device page URL lives. The route
 * therefore carries no device token — only a reason — so there is nothing here
 * to authenticate and nothing to send back to the page they came from.
 */
const DeviceUserSSOErrorPage = ({
  location: { query },
}: WithRouterProps<object, IDeviceUserSSOErrorQuery>) => {
  const isMobileView = useIsMobileWidth();

  return (
    <div className={`${baseClass} app-wrap`}>
      <DeviceUserError
        isMobileView={isMobileView}
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
