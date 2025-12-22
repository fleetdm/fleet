import React from "react";

import { ICommand } from "interfaces/command";

import FeedListItem from "components/FeedListItem";

const baseClass = "command-item";

interface ICommandItemProps {
  command: ICommand;
  onShowDetails: (commandUUID: string, hostUUID: string) => void;
}

const CommandItem = ({ command, onShowDetails }: ICommandItemProps) => {
  const {
    command_status,
    command_uuid,
    host_uuid,
    request_type,
    updated_at,
  } = command;

  let statusVerb = "";
  switch (command_status) {
    case "pending":
      statusVerb = "will run";
      break;
    case "ran":
    case "failed":
      statusVerb = "ran";
      break;
    default:
      statusVerb = "ran";
  }

  const onShowCommandDetails = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onShowDetails(command_uuid, host_uuid);
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
