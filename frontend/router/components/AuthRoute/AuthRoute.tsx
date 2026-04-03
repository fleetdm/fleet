import React, { ReactNode } from "react";
import { useErrorHandler } from "react-error-boundary";

import { useAppContext } from "context/app";

interface IRouteGuardProps {
  children: ReactNode;
}

interface IAuthRouteProps extends IRouteGuardProps {
  permitted: boolean;
}

const AuthRoute = ({ permitted, children }: IAuthRouteProps) => {
  const handlePageError = useErrorHandler();
  const { currentUser } = useAppContext();

  if (!currentUser) {
    return null;
  }

  if (!permitted) {
    handlePageError({ status: 403 });
    return null;
  }

  return <>{children}</>;
};

export const AuthGlobalAdminRoutes = ({ children }: IRouteGuardProps) => {
  const { isGlobalAdmin } = useAppContext();
  return <AuthRoute permitted={!!isGlobalAdmin}>{children}</AuthRoute>;
};

export const AuthAnyAdminRoutes = ({ children }: IRouteGuardProps) => {
  const { isGlobalAdmin, isAnyTeamAdmin } = useAppContext();
  return (
    <AuthRoute permitted={!!isGlobalAdmin || !!isAnyTeamAdmin}>
      {children}
    </AuthRoute>
  );
};

export const AuthGlobalAdminMaintainerRoutes = ({
  children,
}: IRouteGuardProps) => {
  const { isGlobalAdmin, isGlobalMaintainer } = useAppContext();
  return (
    <AuthRoute permitted={!!isGlobalAdmin || !!isGlobalMaintainer}>
      {children}
    </AuthRoute>
  );
};

export const AuthAnyMaintainerAnyAdminRoutes = ({
  children,
}: IRouteGuardProps) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
  } = useAppContext();
  return (
    <AuthRoute
      permitted={
        !!isGlobalAdmin ||
        !!isGlobalMaintainer ||
        !!isAnyTeamAdmin ||
        !!isAnyTeamMaintainer
      }
    >
      {children}
    </AuthRoute>
  );
};

export const AuthAnyMaintainerAdminTechnicianRoutes = ({
  children,
}: IRouteGuardProps) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalTechnician,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
    isAnyTeamTechnician,
  } = useAppContext();
  return (
    <AuthRoute
      permitted={
        !!isGlobalAdmin ||
        !!isGlobalMaintainer ||
        !!isGlobalTechnician ||
        !!isAnyTeamAdmin ||
        !!isAnyTeamMaintainer ||
        !!isAnyTeamTechnician
      }
    >
      {children}
    </AuthRoute>
  );
};

export const AuthAnyMaintainerAdminObserverPlusRoutes = ({
  children,
}: IRouteGuardProps) => {
  const { isAnyMaintainerAdminObserverPlus } = useAppContext();
  return (
    <AuthRoute permitted={!!isAnyMaintainerAdminObserverPlus}>
      {children}
    </AuthRoute>
  );
};

export default AuthRoute;
