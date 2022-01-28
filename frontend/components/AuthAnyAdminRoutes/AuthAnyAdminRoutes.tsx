import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthAnyAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer or any admin when routing
 */
const AuthAnyAdminRoutes = ({
  children,
}: IAuthAnyAdminRoutesProps): JSX.Element | null => {
  const handlePageError = useErrorHandler();
  const { currentUser, isGlobalAdmin, isAnyTeamAdmin } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  if (!isGlobalAdmin && !isAnyTeamAdmin) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthAnyAdminRoutes;
