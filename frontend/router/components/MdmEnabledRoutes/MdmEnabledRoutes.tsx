import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { AppContext } from "context/app";

interface IMdmEnabledRoutesProps {
  children: JSX.Element;
}

const MdmEnabledRoutes = ({ children }: IMdmEnabledRoutesProps) => {
  const handlePageError = useErrorHandler();
  const { isMdmEnabled } = useContext(AppContext);

  if (!isMdmEnabled) {
    handlePageError({ status: 404 });
    return null;
  }
  return <>{children}</>;
};

export default MdmEnabledRoutes;
