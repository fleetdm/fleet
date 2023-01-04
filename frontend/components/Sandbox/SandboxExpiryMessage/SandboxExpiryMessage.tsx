import React from "react";

import ExternalLinkIcon from "../../../../assets/images/icon-external-link-black-12x12@2x.png";

const baseClass = "sandbox-expiry-message";

interface ISandboxExpiryMessageProps {
  expiry: string;
  isNoSandboxHosts?: boolean;
}

const SandboxExpiryMessage = ({
  expiry,
  isNoSandboxHosts,
}: ISandboxExpiryMessageProps) => {
  return (
    <a
      href="https://fleetdm.com/docs/using-fleet/learn-how-to-use-fleet#how-to-add-your-device-to-fleet"
      target="_blank"
      rel="noreferrer"
      className={baseClass}
    >
      <p>Your Fleet Sandbox expires in {expiry}.</p>
      <span>
        {isNoSandboxHosts ? (
          <>
            It&apos;s time to enroll your first host! Navigate to Hosts &gt; Add
            Hosts to get started
          </>
        ) : (
          <>
            Learn how to use Fleet{" "}
            <img alt="Open external link" src={ExternalLinkIcon} />
          </>
        )}
      </span>
    </a>
  );
};

export default SandboxExpiryMessage;
