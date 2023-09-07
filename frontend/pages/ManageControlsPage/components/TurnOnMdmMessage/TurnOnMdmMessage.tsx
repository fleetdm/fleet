import React, { useContext } from "react";
import { AppContext } from "context/app";
import PATHS from "router/paths";

import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";
import { InjectedRouter } from "react-router";

const baseClass = "turn-on-mdm-message";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
}

const TurnOnMdmMessage = ({ router }: ITurnOnMdmMessageProps) => {
  const { isGlobalAdmin } = useContext(AppContext);

  const getInfoText = () => {
    if (isGlobalAdmin) {
      return "Turn on MDM to change settings on your hosts.";
    }
    return "Your Fleet administrator must turn on MDM to change settings on your hosts.";
  };

  const onConnectClick = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  const renderConnectButton = () => {
    if (isGlobalAdmin) {
      return (
        <Button
          variant="brand"
          onClick={onConnectClick}
          className={`${baseClass}__connectAPC-button`}
        >
          Turn on
        </Button>
      );
    }
    return <></>;
  };

  return (
    <EmptyTable
      header="Manage your macOS hosts"
      info={getInfoText()}
      primaryButton={renderConnectButton()}
    />
  );
};

export default TurnOnMdmMessage;
