import classnames from "classnames";
import React, { ReactNode, useContext } from "react";

import LicenseExpirationBanner from "components/LicenseExpirationBanner";
import AndroidEnterpriseDeletedMessage from "components/MDM/AndroidEnterpriseDeletedMessage";
import AppleBMRenewalMessage from "components/MDM/AppleBMRenewalMessage";
import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";
import AppleBMTokenInvalidMessage from "components/MDM/AppleBMTokenInvalidMessage";
import ApplePNCertRenewalMessage from "components/MDM/ApplePNCertRenewalMessage";
import { AppContext } from "context/app";
import { hasLicenseExpired } from "utilities/helpers";

import EJBCACertRenewalMessage from "./banners/EJBCACertRenewalMessage";
import LicenseHostLimitMessage from "./banners/LicenseHostLimitMessage";
import MicrosoftGraphCredentialInvalidMessage from "./banners/MicrosoftGraphCredentialInvalidMessage";
import VppRenewalMessage from "./banners/VppRenewalMessage";

export interface IMainContentConfig {
  renderedBanner: boolean;
}

// Only warn once hosts are more than 20% over the license count.
const LICENSE_HOST_LIMIT_THRESHOLD = 1.2;

interface IMainContentProps {
  children: ReactNode | ((mainContentConfig: IMainContentConfig) => ReactNode);
  /** Whether the page renders its own banner. Page banners take priority over
   * the license host limit banner, so it's hidden when this is true. */
  pageHasBanner?: boolean;
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
  pageHasBanner = false,
}: IMainContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const {
    config,
    hostCount,
    isPremiumTier,
    isAndroidEnterpriseDeleted,
    isApplePnsExpired,
    isAppleBmExpired,
    isVppExpired,
    isEJBCACertExpired,
    needsAbmTermsRenewal,
    hasInvalidABMToken,
    invalidAbmTokenOrgNames,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
    willEJBCACertExpire,
  } = useContext(AppContext);

  const renderAppWideBanner = () => {
    const isFleetLicenseExpired = hasLicenseExpired(
      config?.license.expiration || ""
    );

    const licensedHostCount = config?.license.device_count;

    let banner: JSX.Element | null = null;

    // the order of these checks is important. This is the priority order
    // for showing banners and only one banner is shown at a time.
    if (isApplePnsExpired || willApplePnsExpire) {
      // APNs expiration banner will show for either premium or free tiers
      // but all other banners are only for premium tiers
      banner = <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
    } else if (isPremiumTier) {
      if (isAndroidEnterpriseDeleted) {
        banner = <AndroidEnterpriseDeletedMessage />;
      } else if (isAppleBmExpired || willAppleBmExpire) {
        banner = <AppleBMRenewalMessage expired={isAppleBmExpired} />;
      } else if (needsAbmTermsRenewal) {
        banner = <AppleBMTermsMessage />;
      } else if (hasInvalidABMToken) {
        banner = (
          <AppleBMTokenInvalidMessage orgNames={invalidAbmTokenOrgNames} />
        );
      } else if (isVppExpired || willVppExpire) {
        banner = <VppRenewalMessage expired={isVppExpired} />;
      } else if (isEJBCACertExpired || willEJBCACertExpire) {
        banner = <EJBCACertRenewalMessage expired={isEJBCACertExpired} />;
      } else if (config?.mdm.microsoft_graph_credential_invalid) {
        banner = <MicrosoftGraphCredentialInvalidMessage />;
      } else if (isFleetLicenseExpired) {
        banner = <LicenseExpirationBanner />;
      } else if (
        !pageHasBanner &&
        licensedHostCount &&
        hostCount !== undefined &&
        hostCount > licensedHostCount * LICENSE_HOST_LIMIT_THRESHOLD
      ) {
        banner = (
          <LicenseHostLimitMessage
            hostCount={hostCount}
            licensedHostCount={licensedHostCount}
          />
        );
      }
    }

    if (banner) {
      return (
        <div className={`${baseClass}--animation-disabled`}>
          <div className={`${baseClass}__warning-banner`}>{banner}</div>
        </div>
      );
    }

    return null;
  };

  const appWideBanner = renderAppWideBanner();
  const mainContentConfig: IMainContentConfig = {
    renderedBanner: !!appWideBanner,
  };

  return (
    <div className={classes}>
      {appWideBanner}
      {typeof children === "function" ? children(mainContentConfig) : children}
    </div>
  );
};

export default MainContent;
