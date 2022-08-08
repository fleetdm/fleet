import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthenticatedAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a global admin when routing
 */
const AuthenticatedGlobalAdminRoutes = ({
  children,
}: IAuthenticatedAdminRoutesProps) => {
  const handlePageError = useErrorHandler();
  const { currentUser, isGlobalAdmin } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  if (!isGlobalAdmin) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthenticatedGlobalAdminRoutes;
