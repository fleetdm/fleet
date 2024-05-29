import React, { ReactNode, useContext } from "react";
import classnames from "classnames";
import { formatDistanceToNow } from "date-fns";
import { hasLicenseExpired } from "utilities/helpers";

import SandboxExpiryMessage from "components/Sandbox/SandboxExpiryMessage";
import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";

import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
import SandboxGate from "components/Sandbox/SandboxGate";
import { AppContext } from "context/app";
import LicenseExpirationBanner from "components/LicenseExpirationBanner";

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
    isSandboxMode,
    isPremiumTier,
    noSandboxHosts,
  } = useContext(AppContext);

  const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;
  const isLicenseExpired = hasLicenseExpired(config?.license.expiration || "");

  const sandboxExpiryTime =
    sandboxExpiry === undefined
      ? "..."
      : formatDistanceToNow(new Date(sandboxExpiry));

  const showAppleABMBanner =
    isAppleBmTermsExpired && isPremiumTier && !isSandboxMode;

  // ABM banner takes precedence
  const showLicenseExpirationBanner =
    isLicenseExpired && isPremiumTier && !isAppleBmTermsExpired;

  return (
    <div className={classes}>
      <div className={`${baseClass}--animation-disabled`}>
        {showAppleABMBanner && <AppleBMTermsMessage />}
        {showLicenseExpirationBanner && <LicenseExpirationBanner />}
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
