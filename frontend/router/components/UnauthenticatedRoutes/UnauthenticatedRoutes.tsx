import React from "react";

interface IAppProps {
  children: JSX.Element;
}

export const UnauthenticatedRoutes = ({ children }: IAppProps): JSX.Element => {
  if (window.location.hostname.includes(".sandbox.fleetdm.com")) {
    window.location.href = "https://www.fleetdm.com/try-fleet/login";
  }
  return <div>{children}</div>;
};

export default UnauthenticatedRoutes;
