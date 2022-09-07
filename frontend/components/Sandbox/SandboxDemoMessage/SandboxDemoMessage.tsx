import classnames from "classnames";
import React from "react";

import ExternalLinkIcon from "../../../../assets/images/icon-external-link-12x12@2x.png";

interface ISandboxDemoMessageProps {
  /** message to display in the sandbox error */
  message: string;
  /** UTM (Urchin Tracking Module) source text that is added to the demo link */
  utmSource: string;
  className?: string;
}

const baseClass = "sandbox-demo-message";

const SandboxDemoMessage = ({
  message,
  utmSource,
  className,
}: ISandboxDemoMessageProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <h2 className={`${baseClass}__message`}>{message}</h2>
      <p className={`${baseClass}__link-message`}>
        Want to learn more?
        <a
          href={`https://calendly.com/fleetdm/demo?utm_source=${utmSource}`}
          target="_blank"
          rel="noreferrer"
        >
          Schedule a demo
          <img alt="Open external link" src={ExternalLinkIcon} />
        </a>
      </p>
    </div>
  );
};

export default SandboxDemoMessage;
