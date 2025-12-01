import React, { useContext } from "react";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";
import { InjectedRouter } from "react-router";

const baseClass = "turn-on-message";

interface ITurnOnMessageProps {
  router: InjectedRouter;
  /** @default "Manage your hosts" */
  header?: string;
  /** @default "MDM must be turned on to change settings on your hosts. */
  info?: string;
  /** @default "Turn on" */
  buttonText?: string;
  /** The path to navigate the user to when they press the button.
   * @default PATHS.ADMIN_INTEGRATIONS_MDM */
  path?: string;
}

/** This component renders a message prompting the user to turn on MDM by default. It can also be used as a generic
 * message to prompt the user to take action by providing custom header, info, button text, and path. */
const TurnOnMessage = ({
  router,
  header = "Manage your hosts",
  info = "MDM must be turned on to change settings on your hosts.",
  buttonText = "Turn on",
  path = PATHS.ADMIN_INTEGRATIONS_MDM,
}: ITurnOnMessageProps) => {
  const { isGlobalAdmin } = useContext(AppContext);

  const onConnectClick = () => {
    router.push(path);
  };

  const renderConnectButton = () => {
    return isGlobalAdmin ? (
      <Button
        onClick={onConnectClick}
        className={`${baseClass}__connectAPC-button`}
      >
        {buttonText}
      </Button>
    ) : (
      <></>
    );
  };

  return (
    <EmptyTable
      className={baseClass}
      header={header}
      info={info}
      primaryButton={renderConnectButton()}
    />
  );
};

export default TurnOnMessage;
