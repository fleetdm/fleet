import React from "react";

import { ICommand } from "interfaces/command";

import Pagination from "components/Pagination";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import CommandItem, {
  ShowCommandDetailsHandler,
} from "../CommandItem/CommandItem";

const baseClass = "past-command-feed";

interface IPastCommandFeedProps {
  commands: ICommand[];
  onShowDetails: ShowCommandDetailsHandler;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const PastCommandFeed = ({
  commands,
  onShowDetails,
  onNextPage,
  onPreviousPage,
}: IPastCommandFeedProps) => {
  if (commands.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message="Completed MDM commands will appear here."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div>
        {commands.map((command: ICommand) => {
          return (
            <CommandItem
              key={`${command.command_uuid}+${command.host_uuid}`}
              command={command}
              onShowDetails={onShowDetails}
            />
          );
        })}
      </div>
      {/* <Pagination */}
      {/*   disablePrev={!meta.has_previous_results} */}
      {/*   disableNext={!meta.has_next_results} */}
      {/*   hidePagination={!meta.has_next_results && !meta.has_previous_results} */}
      {/*   onPrevPage={onPreviousPage} */}
      {/*   onNextPage={onNextPage} */}
      {/* /> */}
    </div>
  );
};

export default PastCommandFeed;
