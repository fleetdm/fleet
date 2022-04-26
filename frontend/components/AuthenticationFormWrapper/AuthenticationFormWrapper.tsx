import React from "react";

// @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

interface IAuthenticationFormWrapperProps {
  children: React.ReactNode;
}

const baseClass = "auth-form-wrapper";

const AuthenticationFormWrapper = ({
  children,
}: IAuthenticationFormWrapperProps) => (
  <div className={baseClass}>
    <img alt="Fleet" src={fleetLogoText} className={`${baseClass}__logo`} />
    {children}
  </div>
);

export default AuthenticationFormWrapper;
