import React from "react";

import { ICommand } from "interfaces/command";

import FeedListItem from "components/FeedListItem";

const baseClass = "command-item";

/**
 * Handler that will show the details of a command. This is used to pass
 * the details of a command to the parent component to show the details of
 * the command.
 */
export type ShowCommandDetailsHandler = (cmd: ICommand) => void;

interface ICommandItemProps {
  command: ICommand;
  onShowDetails: ShowCommandDetailsHandler;
}

const CommandItem = ({ command, onShowDetails }: ICommandItemProps) => {
  const { command_status, request_type, updated_at } = command;

  let statusVerb = "";
  switch (command_status) {
    case "pending":
      statusVerb = "will run";
      break;
    case "ran":
    case "failed":
      statusVerb = command_status;
      break;
    default:
      statusVerb = "ran";
  }

  const onShowCommandDetails = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onShowDetails(command);
  };

  return (
    <FeedListItem
      className={baseClass}
      useFleetAvatar
      allowShowDetails
      createdAt={new Date(updated_at)}
      onClickFeedItem={onShowCommandDetails}
    >
      The <b>{request_type}</b> command {statusVerb}.
    </FeedListItem>
  );
};

export default CommandItem;
