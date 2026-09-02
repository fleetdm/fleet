import React from "react";

import { ICommand, isCancelableCommand } from "interfaces/command";
import { IGetCommandsResponse } from "services/entities/command";

import Pagination from "components/Pagination";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import CommandItem, {
  CancelCommandHandler,
  ShowCommandDetailsHandler,
} from "../CommandItem/CommandItem";

const baseClass = "command-feed";

interface ICommandFeedProps {
  commands: IGetCommandsResponse;
  emptyDescription: string;
  onShowDetails: ShowCommandDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
  /** When provided, cancelable pending commands render a cancel button. */
  onCancelCommand?: CancelCommandHandler;
}

const CommandFeed = ({
  commands,
  emptyDescription,
  onShowDetails,
  onNextPage,
  onPreviousPage,
  onCancelCommand,
}: ICommandFeedProps) => {
  const { meta, results } = commands;
  if (results === null || results.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message={emptyDescription}
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {results.map((command: ICommand) => {
          return (
            <CommandItem
              key={`${command.command_uuid}+${command.host_uuid}`}
              command={command}
              onShowDetails={onShowDetails}
              onCancel={
                onCancelCommand && isCancelableCommand(command)
                  ? onCancelCommand
                  : undefined
              }
            />
          );
        })}
      </div>
      <Pagination
        disablePrev={!meta.has_previous_results}
        disableNext={!meta.has_next_results}
        hidePagination={!meta.has_next_results && !meta.has_previous_results}
        onPrevPage={onPreviousPage}
        onNextPage={onNextPage}
      />
    </div>
  );
};

export default CommandFeed;
