import React from "react";

import ExternalURLIcon from "../../../../assets/images/icon-external-url-black-12x12@2x.png";

const baseClass = "sandbox-expiry-message";

interface ISandboxExpiryMessageProps {
  expiry: string;
}

const SandboxExpiryMessage = ({ expiry }: ISandboxExpiryMessageProps) => {
  return (
    <a
      href="https://fleetdm.com/docs/deploying"
      target="_blank"
      rel="noreferrer"
      className={baseClass}
    >
      <p>Your Fleet Sandbox expires in {expiry}.</p>
      <span>
        Learn how to deploy Fleet
        <img
          alt="Open external link"
          className="icon-external"
          src={ExternalURLIcon}
        />
      </span>
    </a>
  );
};

export default SandboxExpiryMessage;
