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
    apnsExpiry,
    abmExpiry,
  } = useContext(AppContext);

  const sandboxExpiryTime =
    sandboxExpiry === undefined
      ? "..."
      : formatDistanceToNow(new Date(sandboxExpiry));

  const renderAppWideBanner = () => {
    const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;
    const isApplePnsExpired = hasLicenseExpired(apnsExpiry || "");
    const willApplePnsExpireIn30Days = willExpireWithinXDays(
      apnsExpiry || "",
      30
    );
    const isAppleBmExpired = hasLicenseExpired(abmExpiry || ""); // NOTE: See Rachel's related FIXME added to App.tsx in https://github.com/fleetdm/fleet/pull/19571
    const willAppleBmExpireIn30Days = willExpireWithinXDays(
      abmExpiry || "",
      30
    );
    const isFleetLicenseExpired = hasLicenseExpired(
      config?.license.expiration || ""
    );

    if (isPremiumTier) {
      if (isApplePnsExpired || willApplePnsExpireIn30Days) {
        return <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
      }

      if (isAppleBmExpired || willAppleBmExpireIn30Days) {
        return <AppleBMRenewalMessage expired={isAppleBmExpired} />;
      }

      if (isAppleBmTermsExpired) {
        return <AppleBMTermsMessage />;
      }

      if (isFleetLicenseExpired) {
        return <LicenseExpirationBanner />;
      }
    }

    return <></>;
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
