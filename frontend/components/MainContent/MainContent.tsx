import React, { ReactNode, useContext } from "react";
import classnames from "classnames";
import { formatDistanceToNow } from "date-fns";
import { hasLicenseExpired, willExpireWithinXDays } from "utilities/helpers";

import SandboxExpiryMessage from "components/Sandbox/SandboxExpiryMessage";
import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";

import SandboxGate from "components/Sandbox/SandboxGate";
import { AppContext } from "context/app";
import LicenseExpirationBanner from "components/LicenseExpirationBanner";
import ApplePNCertRenewalMessage from "components/MDM/ApplePNCertRenewalMessage";
import AppleBMRenewalMessage from "components/MDM/AppleBMRenewalMessage";
import VppRenewalMessage from "./banners/VppRenewalMessage";

interface IMainContentProps {
  children: ReactNode;
  /** An optional classname to pass to the main content component.
   * This can be used to apply styles directly onto the main content div
   */
  className?: string;
}

const baseClass = "main-content";

/**
 * A component that controls the layout and styling of the main content region
 * of the application.
 */
const MainContent = ({
  children,
  className,
}: IMainContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const {
    sandboxExpiry,
    config,
    isPremiumTier,
    noSandboxHosts,
    isApplePnsExpired,
    isAppleBmExpired,
    isVppExpired,
    needsAbmTermsRenewal,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
  } = useContext(AppContext);

  const sandboxExpiryTime =
    sandboxExpiry === undefined
      ? "..."
      : formatDistanceToNow(new Date(sandboxExpiry));

  const renderAppWideBanner = () => {
    const isFleetLicenseExpired = hasLicenseExpired(
      config?.license.expiration || ""
    );

    let banner: JSX.Element | null = null;

    if (isPremiumTier) {
      if (isApplePnsExpired || willApplePnsExpire) {
        banner = <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
      } else if (isAppleBmExpired || willAppleBmExpire) {
        banner = <AppleBMRenewalMessage expired={isAppleBmExpired} />;
      } else if (needsAbmTermsRenewal) {
        banner = <AppleBMTermsMessage />;
      } else if (isVppExpired || willVppExpire) {
        banner = <VppRenewalMessage expired={isVppExpired} />;
      } else if (isFleetLicenseExpired) {
        banner = <LicenseExpirationBanner />;
      }
    }

    if (banner) {
      return <div className={`${baseClass}__warning-banner`}>{banner}</div>;
    }

    return null;
  };
  return (
    <div className={classes}>
      <div className={`${baseClass}--animation-disabled`}>
        {renderAppWideBanner()}
        <SandboxGate
          fallbackComponent={() => (
            <SandboxExpiryMessage
              expiry={sandboxExpiryTime}
              noSandboxHosts={noSandboxHosts}
            />
          )}
        />
      </div>
      {children}
    </div>
  );
};

export default MainContent;
