import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthenticatedAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer or any admin when routing
 */
const AuthenticatedAdminRoutes = ({
  children,
}: IAuthenticatedAdminRoutesProps): JSX.Element | null => {
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

export default AuthenticatedAdminRoutes;
