import classnames from "classnames";
import React from "react";

import CustomLink from "components/CustomLink";

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
        Want to learn more?{" "}
        <CustomLink
          url={`https://calendly.com/fleetdm/demo?utm_source=${utmSource}`}
          text={"Schedule a demo"}
          newTab
        />
      </p>
    </div>
  );
};

export default SandboxDemoMessage;
