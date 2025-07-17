import React, { ReactNode, useContext } from "react";
import classnames from "classnames";

import { hasLicenseExpired } from "utilities/helpers";
import { AppContext } from "context/app";

import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";
import LicenseExpirationBanner from "components/LicenseExpirationBanner";
import ApplePNCertRenewalMessage from "components/MDM/ApplePNCertRenewalMessage";
import AppleBMRenewalMessage from "components/MDM/AppleBMRenewalMessage";
import AndroidEnterpriseDeletedMessage from "components/MDM/AndroidEnterpriseDeletedMessage";

import VppRenewalMessage from "./banners/VppRenewalMessage";

export interface IMainContentConfig {
  renderedBanner: boolean;
}

interface IMainContentProps {
  children?: ReactNode;
  /** An optional classname to pass to the main content component.
   * This can be used to apply styles directly onto the main content div
   */
  className?: string;
  renderChildren?: (mainContentConfig: IMainContentConfig) => ReactNode;
}

const baseClass = "main-content";

/**
 * A component that controls the layout and styling of the main content region
 * of the application.
 */
const MainContent = ({
  children,
  className,
  renderChildren,
}: IMainContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const {
    config,
    isPremiumTier,
    isAndroidEnterpriseDeleted,
    isApplePnsExpired,
    isAppleBmExpired,
    isVppExpired,
    needsAbmTermsRenewal,
    willAppleBmExpire,
    willApplePnsExpire,
    willVppExpire,
  } = useContext(AppContext);

  const renderAppWideBanner = () => {
    const isFleetLicenseExpired = hasLicenseExpired(
      config?.license.expiration || ""
    );

    let banner: JSX.Element | null = null;

    // the order of these checks is important. This is the priority order
    // for showing banners and only one banner is shown at a time.
    if (isPremiumTier) {
      if (isApplePnsExpired || willApplePnsExpire) {
        banner = <ApplePNCertRenewalMessage expired={isApplePnsExpired} />;
      } else if (false) {
        // TODO: remove this when API is ready
        banner = <AndroidEnterpriseDeletedMessage />;
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
      {renderChildren ? renderChildren(mainContentConfig) : children}
    </div>
  );
};

export default MainContent;
