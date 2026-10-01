import React from "react";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { IConfig } from "interfaces/config";
import SettingsSection from "pages/admin/components/SettingsSection";

import GoogleWorkspaceSection from "./components/GoogleWorkspaceSection";
import IdentityProviderSection from "./components/IdentityProviderSection";

const baseClass = "identity-providers";

interface IIdentityProvidersProps {
  appConfig: IConfig;
  isPremiumTier: boolean;
}

const IdentityProviders = ({
  appConfig,
  isPremiumTier,
}: IIdentityProvidersProps) => {
  // Both sections are premium-only, so gate them here once rather than in each
  // child. Keep the section title above the message to match other settings
  // sections' free-tier pattern.
  if (!isPremiumTier) {
    return (
      <div className={baseClass}>
        <SettingsSection title="Identity provider (IdP)">
          <PremiumFeatureMessage />
        </SettingsSection>
      </div>
    );
  }

  return (
    <div className={baseClass}>
      <IdentityProviderSection />
      <GoogleWorkspaceSection appConfig={appConfig} />
    </div>
  );
};

export default IdentityProviders;
