import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthAnyMaintainerAdminObserverPlusRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer, admin, or observer plus when routing
 */
const AuthAnyMaintainerAdminObserverPlusRoutes = ({
  children,
}: IAuthAnyMaintainerAdminObserverPlusRoutesProps) => {
  const handlePageError = useErrorHandler();
  const { currentUser, isAnyMaintainerAdminObserverPlus } = useContext(
    AppContext
  );

  if (!currentUser) {
    return null;
  }

  if (!isAnyMaintainerAdminObserverPlus) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthAnyMaintainerAdminObserverPlusRoutes;
