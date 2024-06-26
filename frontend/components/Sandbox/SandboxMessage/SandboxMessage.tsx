import classnames from "classnames";
import React from "react";

import CustomLink from "components/CustomLink";

interface ISandboxMessageProps {
  variant?: "demo" | "sales";
  /** message to display in the sandbox error */
  message: string;
  /** UTM (Urchin Tracking Module) source text that is added to the demo link */
  utmSource?: string;
  className?: string;
}

const baseClass = "sandbox-message";

const SandboxMessage = ({
  variant = "demo",
  message,
  utmSource,
  className,
}: ISandboxMessageProps): JSX.Element => {
  const classes = classnames(baseClass, className);
  const variants = {
    demo: (
      <CustomLink
        url={`https://calendly.com/fleetdm/demo?utm_source=${utmSource}`}
        text="Schedule a demo"
        newTab
      />
    ),
    sales: (
      <CustomLink
        url={`https://fleetdm.com/upgrade`}
        text="Contact sales"
        newTab
      />
    ),
  };

  return (
    <div className={classes}>
      <h2 className={`${baseClass}__message`}>{message}</h2>
      <p className={`${baseClass}__link-message`}>
        Want to learn more? {variants[variant]}
      </p>
    </div>
  );
};

export default SandboxMessage;
