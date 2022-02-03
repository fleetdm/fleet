import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthGlobalAdminMaintainerRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer or any admin when routing
 */
const AuthGlobalAdminMaintainerRoutes = ({
  children,
}: IAuthGlobalAdminMaintainerRoutesProps): JSX.Element | null => {
  const handlePageError = useErrorHandler();
  const { currentUser, isGlobalAdmin, isGlobalMaintainer } = useContext(
    AppContext
  );

  if (!currentUser) {
    return null;
  }

  if (!isGlobalAdmin && !isGlobalMaintainer) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthGlobalAdminMaintainerRoutes;
