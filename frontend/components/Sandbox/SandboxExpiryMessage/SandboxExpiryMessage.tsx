import React from "react";
import { browserHistory } from "react-router";
import PATHS from "router/paths";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "sandbox-expiry-message";

interface ISandboxExpiryMessageProps {
  expiry: string;
  noSandboxHosts?: boolean;
}

const SandboxExpiryMessage = ({
  expiry,
  noSandboxHosts,
}: ISandboxExpiryMessageProps) => {
  const openAddHostModal = () => {
    browserHistory.push(PATHS.MANAGE_HOSTS_ADD_HOSTS);
  };

  if (noSandboxHosts) {
    return (
      <div className={baseClass}>
        <p>Your Fleet Sandbox expires in {expiry}.</p>
        <div className={`${baseClass}__tip`}>
          <Icon name="lightbulb" size="large" />
          <p>
            <b>Quick tip: </b> Enroll a host to get started.
          </p>
          <form>
            <Button
              onClick={openAddHostModal}
              className={`${baseClass}__add-hosts`}
              variant="brand"
            >
              <span>Add hosts</span>
            </Button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <a
      href="https://fleetdm.com/docs/using-fleet/learn-how-to-use-fleet#how-to-add-your-device-to-fleet"
      target="_blank"
      rel="noreferrer"
      className={baseClass}
    >
      <p>Your Fleet Sandbox expires in {expiry}.</p>
      <p>
        <b>Learn how to use Fleet</b>{" "}
        <Icon name="external-link" color="core-fleet-black" />
      </p>
    </a>
  );
};

export default SandboxExpiryMessage;
