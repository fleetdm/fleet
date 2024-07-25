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
    apnsExpiry = "",
    abmExpiry = "",
    vppExpiry = "",
  } = useContext(AppContext);

  const sandboxExpiryTime =
    sandboxExpiry === undefined
      ? "..."
      : formatDistanceToNow(new Date(sandboxExpiry));

  const renderAppWideBanner = () => {
    const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;
    const isApplePnsExpired = hasLicenseExpired(apnsExpiry);
    const willApplePnsExpireIn30Days = willExpireWithinXDays(apnsExpiry, 30);
    const isAppleBmExpired = hasLicenseExpired(abmExpiry); // NOTE: See Rachel's related FIXME added to App.tsx in https://github.com/fleetdm/fleet/pull/19571
    const willAppleBmExpireIn30Days = willExpireWithinXDays(abmExpiry, 30);
    const isFleetLicenseExpired = hasLicenseExpired(
      config?.license.expiration || ""
    );

    const isVppExpired = hasLicenseExpired(vppExpiry);
    const willVppExpireIn30Days = willExpireWithinXDays(vppExpiry, 30);

    let banner: JSX.Element | null = null;

    if (isPremiumTier) {
      if (isApplePnsExpired || willApplePnsExpireIn30Days) {
        banner = <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
      } else if (isAppleBmExpired || willAppleBmExpireIn30Days) {
        banner = <AppleBMRenewalMessage expired={isAppleBmExpired} />;
      } else if (isAppleBmTermsExpired) {
        banner = <AppleBMTermsMessage />;
      } else if (isFleetLicenseExpired) {
        banner = <LicenseExpirationBanner />;
      } else if (isVppExpired) {
        banner = <VppRenewalMessage expired={willVppExpireIn30Days} />;
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
