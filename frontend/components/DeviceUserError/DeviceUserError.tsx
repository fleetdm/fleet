import React from "react";

import classNames from "classnames";
import Icon from "components/Icon/Icon";
import DataError from "components/DataError";

const baseClass = "device-user-error";

/** Why the Fleet Desktop SSO flow left the end user without a session. */
export type DeviceSSOErrorReason =
  /** The handshake ran out before the IdP posted back. */
  | "session_expired"
  /** The IdP posted back, but the callback couldn't mint a session. */
  | "callback_failed"
  /** Fleet couldn't start the flow at all. */
  | "initiate_failed"
  /** The round-trip finished and still left no session behind. */
  | "not_signed_in";

const SSO_ERROR_COPY: Record<
  DeviceSSOErrorReason,
  { header: string; description: string }
> = {
  session_expired: {
    header: "Your sign-in session expired.",
    description: "Open Fleet Desktop and click “My device” to try again.",
  },
  callback_failed: {
    header: "Couldn't finish signing in.",
    description: "Open Fleet Desktop and click “My device” to try again.",
  },
  initiate_failed: {
    header: "Couldn't start signing in.",
    description:
      "Your organization requires single sign-on to view this page. Try again, or contact your IT admin.",
  },
  not_signed_in: {
    header: "Couldn't sign in.",
    description:
      "Your organization requires single sign-on to view this page. Try again, or contact your IT admin.",
  },
};

interface IDeviceUserErrorProps {
  /** Modifies styling for mobile width (<768px) */
  isMobileView?: boolean;
  /** Modifies error message for iPhone/iPad/Android */
  isMobileDevice?: boolean;
  isAuthenticationError?: boolean;
  isErrorSetupSteps?: boolean;
  /** Renders Fleet Desktop SSO failure copy instead of the generic message. */
  ssoError?: DeviceSSOErrorReason;
  /** Rendered below the description, e.g. a "Sign in again" button. */
  action?: React.ReactNode;
}

const DeviceUserError = ({
  isMobileView = false,
  isMobileDevice = false,
  isAuthenticationError = false,
  isErrorSetupSteps = false,
  ssoError,
  action,
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

  if (ssoError) {
    const { header, description } = SSO_ERROR_COPY[ssoError];
    headerContent = (
      <>
        <Icon name="error" />
        {header}
      </>
    );
    descriptionContent = description;
  } else if (isAuthenticationError) {
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
        To access your device information, please click “My Device” from the
        Fleet Desktop menu icon.
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
          {action && <div className={`${baseClass}__action`}>{action}</div>}
        </div>
      </div>
    </div>
  );
};

export default DeviceUserError;
