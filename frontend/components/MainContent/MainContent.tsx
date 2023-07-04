import React, { ReactChild, useContext } from "react";
import classnames from "classnames";
import { formatDistanceToNow } from "date-fns";

import SandboxExpiryMessage from "components/Sandbox/SandboxExpiryMessage";
import AppleBMTermsMessage from "components/MDM/AppleBMTermsMessage";

import SandboxGate from "components/Sandbox/SandboxGate";
import { AppContext } from "context/app";

interface IMainContentProps {
  children: ReactChild;
  /** An optional classname to pass to the main content component.
   * This can be used to apply styles directly onto the main content div
   */
  className?: string;
}

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
  const {
    sandboxExpiry,
    config,
    isSandboxMode,
    isPremiumTier,
    noSandboxHosts,
  } = useContext(AppContext);

  const isAppleBmTermsExpired = config?.mdm?.apple_bm_terms_expired;

  const sandboxExpiryTime =
    sandboxExpiry === undefined
      ? "..."
      : formatDistanceToNow(new Date(sandboxExpiry));

  return (
    <div className={classes}>
      {isAppleBmTermsExpired && isPremiumTier && !isSandboxMode && (
        <AppleBMTermsMessage />
      )}
      <SandboxGate
        fallbackComponent={() => (
          <SandboxExpiryMessage
            expiry={sandboxExpiryTime}
            noSandboxHosts={noSandboxHosts}
          />
        )}
      />
      {children}
    </div>
  );
};

export default MainContent;
