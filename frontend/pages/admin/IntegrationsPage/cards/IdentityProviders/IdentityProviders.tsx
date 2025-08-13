import React from "react";

import IdentityProviderSection from "./components/IdentityProviderSection";
import EndUserAuthSection from "./components/EndUserAuthSection";

const baseClass = "identity-providers";

const IdentityProviders = () => {
  return (
    <div className={baseClass}>
      <IdentityProviderSection />
      <EndUserAuthSection />
    </div>
  );
};

export default IdentityProviders;
