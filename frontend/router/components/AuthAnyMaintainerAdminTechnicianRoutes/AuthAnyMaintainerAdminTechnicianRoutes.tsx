import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IAuthAnyMaintainerAdminTechnicianRoutesProps {
  children: JSX.Element;
}

/**
 * Checks if a user is any maintainer, any admin, or any technician when routing
 */
const AuthAnyMaintainerAdminTechnicianRoutes = ({
  children,
}: IAuthAnyMaintainerAdminTechnicianRoutesProps) => {
  const handlePageError = useErrorHandler();
  const {
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalTechnician,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
    isAnyTeamTechnician,
  } = useContext(AppContext);

  if (!currentUser) {
    return null;
  }

  if (
    !isGlobalAdmin &&
    !isGlobalMaintainer &&
    !isGlobalTechnician &&
    !isAnyTeamAdmin &&
    !isAnyTeamMaintainer &&
    !isAnyTeamTechnician
  ) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export default AuthAnyMaintainerAdminTechnicianRoutes;
