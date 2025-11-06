import React from "react";

import IdentityProviderSection from "./components/IdentityProviderSection";

const baseClass = "identity-providers";

const IdentityProviders = () => {
  return (
    <div className={baseClass}>
      <IdentityProviderSection />
    </div>
  );
};

export default IdentityProviders;
