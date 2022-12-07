import React from "react";

import Icon from "components/Icon";

const baseClass = "sandbox-expiry-message";

interface ISandboxExpiryMessageProps {
  expiry: string;
}

// TODO: Check spacing
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
        <Icon name="external-link" />
      </span>
    </a>
  );
};

export default SandboxExpiryMessage;
