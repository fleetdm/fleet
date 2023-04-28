import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IExcludeInSandboxRoutesProps {
  children: JSX.Element;
}

const ExcludeInSandboxRoutes = ({ children }: IExcludeInSandboxRoutesProps) => {
  const handlePageError = useErrorHandler();
  const { isSandboxMode } = useContext(AppContext);

  if (isSandboxMode) {
    handlePageError({ status: 403 });
    return null;
  }
  return <>{children}</>;
};

export default ExcludeInSandboxRoutes;
