import React, { ReactNode, useContext } from "react";

import { AppContext } from "context/app";

interface ISandboxGateProps {
  children?: ReactNode;
  /** The component rendered if the use is in sandbox mode */
  fallbackComponent?: () => ReactNode;
}

/**
 * Checks for and conditionally renders children content depending on a sandbox
 * mode check
 */
const SandboxGate = ({
  children,
  fallbackComponent = () => null,
}: ISandboxGateProps): JSX.Element => {
  const { isSandboxMode } = useContext(AppContext);

  return <>{isSandboxMode ? fallbackComponent() : <>{children}</>}</>;
};

export default SandboxGate;
