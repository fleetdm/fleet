import React from "react";

import ExternalURLIcon from "../../../../assets/images/icon-external-url-black-12x12@2x.png";

const baseClass = "sandbox-expiry-message";

const SandboxExpiryMessage = (): JSX.Element => {
  return (
    <div className={baseClass}>
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

export default SandboxExpiryMessage;
