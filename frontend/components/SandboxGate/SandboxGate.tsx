import React, { ReactNode, useContext } from "react";

import { AppContext } from "context/app";
import ExternalURLIcon from "../../../assets/images/icon-external-url-12x12@2x.png";

interface ISandboxErrorMessageProps {
  message: string;
  utmSource: string;
}

const baseClass = "sandbox-error-message";

const SandboxErrorMessage = ({
  message,
  utmSource,
}: ISandboxErrorMessageProps) => {
  return (
    <div className={baseClass}>
      <h2 className={`${baseClass}__message`}>{message}</h2>
      <p className={`${baseClass}__link-message`}>
        Want to learn more?
        <a
          href={`https://calendly.com/fleetdm/demo?utm_source=${utmSource}`}
          target="_blank"
          rel="noreferrer"
        >
          Schedule a demo
        </a>
        <img
          alt="Open external link"
          className="icon-external"
          src={ExternalURLIcon}
        />
      </p>
    </div>
  );
};

interface ISandboxGateProps {
  /** message to display in the sandbox error */
  message: string;
  /** UTM (Urchin Tracking Module) source text that is added to the demo link */
  utmSource: string;
  children: ReactNode;
}

/**
 * Checks for and conditionally renders children content depending on a sandbox
 * mode check
 */
const SandboxGate = ({
  message,
  utmSource,
  children,
}: ISandboxGateProps): JSX.Element => {
  const { isSandboxMode } = useContext(AppContext);

  return (
    <>
      {isSandboxMode ? (
        <SandboxErrorMessage message={message} utmSource={utmSource} />
      ) : (
        <>{children}</>
      )}
    </>
  );
};

export default SandboxGate;
