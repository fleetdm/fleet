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

const getStatusText = (command: ICommand): string => {
  const { command_status, status } = command;

  // Differentiate NotNow from regular Pending
  if (status === "NotNow") {
    return "is deferred";
  }

  switch (command_status) {
    case "pending":
      return "is pending";
    case "failed":
      return "failed";
    case "ran":
    default:
      return "was acknowledged";
  }
};

const CommandItem = ({ command, onShowDetails }: ICommandItemProps) => {
  const { request_type, updated_at, name, status } = command;

  const statusText = getStatusText(command);
  const willRetryText = status === "NotNow" ? " Fleet will try again." : "";

  const onShowCommandDetails = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onShowDetails(command);
  };

  const activityText = name ? (
    <>
      The <b>{request_type}</b> command for <b>{name}</b> {statusText}.
      {willRetryText}
    </>
  ) : (
    <>
      The <b>{request_type}</b> command {statusText}.{willRetryText}
    </>
  );

  return (
    <FeedListItem
      className={baseClass}
      useFleetAvatar
      allowShowDetails
      createdAt={new Date(updated_at)}
      onClickFeedItem={onShowCommandDetails}
    >
      {activityText}
    </FeedListItem>
  );
};

export default CommandItem;
