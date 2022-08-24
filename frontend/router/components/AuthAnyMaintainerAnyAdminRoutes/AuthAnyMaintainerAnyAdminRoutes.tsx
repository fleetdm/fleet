import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthAnyMaintainerAnyAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer or any admin when routing
 */
const AuthAnyMaintainerAnyAdminRoutes = ({
  children,
}: IAuthAnyMaintainerAnyAdminRoutesProps) => {
  const handlePageError = useErrorHandler();
  const {
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
  } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  if (
    !isGlobalAdmin &&
    !isGlobalMaintainer &&
    !isAnyTeamAdmin &&
    !isAnyTeamMaintainer
  ) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthAnyMaintainerAnyAdminRoutes;
