import React from "react";

import { ICommand } from "interfaces/command";
import { IGetCommandsResponse } from "services/entities/command";

import Pagination from "components/Pagination";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import CommandItem, {
  ShowCommandDetailsHandler,
} from "../CommandItem/CommandItem";

const baseClass = "command-feed";

interface ICommandFeedProps {
  commands: IGetCommandsResponse;
  emptyDescription: string;
  onShowDetails: ShowCommandDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const CommandFeed = ({
  commands,
  emptyDescription,
  onShowDetails,
  onNextPage,
  onPreviousPage,
}: ICommandFeedProps) => {
  const { meta, results } = commands;
  if (results === null || results.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message={description}
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
