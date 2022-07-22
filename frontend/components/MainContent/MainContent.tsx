import classnames from "classnames";
import React, { ReactChild } from "react";

import ExternalURLIcon from "../../../assets/images/icon-external-url-black-12x12@2x.png";

interface IMainContentProps {
  children: ReactChild;
  /** An optional classname to pass to the main content component.
   * This can be used to apply styles directly onto the main content div
   */
  className?: string;
}

const messageClassName = "sandbox-expiry-message";

const SandboxExpiryMessage = (): JSX.Element => {
  return (
    <div className={messageClassName}>
      <p>Your Fleet Sandbox Expires in about 20 hours.</p>
      <a
        href="https://fleetdm.com/docs/deploying"
        target="_blank"
        rel="noreferrer"
      >
        Learn how to renew or downgrade
        <img
          alt="Open external link"
          className="icon-external"
          src={ExternalURLIcon}
        />
      </a>
    </div>
  );
};

const baseClass = "main-content";

/**
 * A component that controls the layout and styling of the main content region
 * of the application.
 */
const MainContent = ({
  children,
  className,
}: IMainContentProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <SandboxExpiryMessage />
      {children}
    </div>
  );
};

export default MainContent;
