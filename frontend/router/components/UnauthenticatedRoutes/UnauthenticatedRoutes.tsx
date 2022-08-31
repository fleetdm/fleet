import React from "react";

interface IAppProps {
  children: JSX.Element;
}

export const UnauthenticatedRoutes = ({ children }: IAppProps): JSX.Element => {
  return <div>{children}</div>;
};

export default UnauthenticatedRoutes;
