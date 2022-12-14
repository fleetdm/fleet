import React, { ReactNode, useContext } from "react";

import { AppContext } from "context/app";

interface IAppleBMTermsGateProps {
  children?: ReactNode;
  /** The component rendered if the use is in sandbox mode */
  fallbackComponent?: () => ReactNode;
}

/**
 * Checks for and conditionally renders children content depending on a sandbox
 * mode check
 */
const AppleBMTermsGate = ({
  children,
  fallbackComponent = () => null,
}: IAppleBMTermsGateProps): JSX.Element => {
  const { config } = useContext(AppContext);

  return (
    <>
      {config?.mdm.apple_bm_terms_expired ? (
        fallbackComponent()
      ) : (
        <>{children}</>
      )}
    </>
  );
};

export default AppleBMTermsGate;
