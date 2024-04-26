import React from "react";
import PATHS from "router/paths";

import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";
import { InjectedRouter } from "react-router";

const baseClass = "turn-on-mdm-message";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
}

const TurnOnMdmMessage = ({ router }: ITurnOnMdmMessageProps) => {
  const onConnectClick = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  const renderConnectButton = () => {
    return (
      <Button
        variant="brand"
        onClick={onConnectClick}
        className={`${baseClass}__connectAPC-button`}
      >
        Turn on
      </Button>
    );
  };

  return (
    <EmptyTable
      header="Manage your hosts"
      info="Turn on MDM to change settings on your hosts."
      primaryButton={renderConnectButton()}
    />
  );
};

export default TurnOnMdmMessage;
