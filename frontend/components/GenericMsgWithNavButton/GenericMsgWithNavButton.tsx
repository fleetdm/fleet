import React, { useContext } from "react";

import { AppContext } from "context/app";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";
import { InjectedRouter } from "react-router";

const baseClass = "generic-msg-with-nav-button";

interface IGenericMsgWithNavButtonProps {
  router: InjectedRouter;
  header: string;
  info: string;
  /** The path to navigate the user to when they press the button. */
  path: string;
  buttonText: string;
}

/** This is a generic component that renders a message with a header, info, and button that will navigate to a path
 * for global admins
 *
 * TODO: consider removing isGlobalAdmin check in here and pushing up to parent */
const GenericMsgWithNavButton = ({
  router,
  header,
  info,
  path,
  buttonText,
}: IGenericMsgWithNavButtonProps) => {
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

export default GenericMsgWithNavButton;
