import React from "react";

import { ICommand } from "interfaces/command";

import EmptyFeed from "../EmptyFeed/EmptyFeed";
import CommandItem from "../CommandItem/CommandItem";

const baseClass = "upcoming-command-feed";

interface IUpcomingCommandFeedProps {
  commands: ICommand[];
  onShowDetails: (commandUUID: string, hostUUID: string) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
}

const UpcomingCommandFeed = ({
  commands,
  onShowDetails,
  onNextPage,
  onPreviousPage,
}: IUpcomingCommandFeedProps) => {
  if (commands.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message="Pending MDM commands will appear here."
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

export default UpcomingCommandFeed;
