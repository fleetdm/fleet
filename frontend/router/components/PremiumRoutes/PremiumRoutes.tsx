import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthenticatedAdminRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if on the premium tier routing
 */
const PremiumRoutes = ({ children }: IAuthenticatedAdminRoutesProps) => {
  const handlePageError = useErrorHandler();
  const { currentUser, isPremiumTier } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  if (!isPremiumTier) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default PremiumRoutes;
