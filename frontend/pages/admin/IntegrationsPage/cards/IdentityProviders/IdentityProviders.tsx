import React from "react";

import { IConfig } from "interfaces/config";

import IdentityProviderSection from "./components/IdentityProviderSection";
import GoogleWorkspaceSection from "./components/GoogleWorkspaceSection";

const baseClass = "identity-providers";

interface IIdentityProvidersProps {
  appConfig: IConfig;
}

const IdentityProviders = ({ appConfig }: IIdentityProvidersProps) => {
  return (
    <div className={baseClass}>
      <IdentityProviderSection />
      <GoogleWorkspaceSection appConfig={appConfig} />
    </div>
  );
};

export default IdentityProviders;
