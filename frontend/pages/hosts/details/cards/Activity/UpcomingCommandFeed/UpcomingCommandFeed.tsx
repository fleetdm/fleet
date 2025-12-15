import React from "react";

import { ICommand } from "interfaces/command";

import EmptyFeed from "../EmptyFeed/EmptyFeed";

const baseClass = "upcoming-command-feed";

interface IUpcomingCommandFeedProps {
  commands: ICommand[];
}

const UpcomingCommandFeed = ({ commands }: IUpcomingCommandFeedProps) => {
  if (commands.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message="Pending MDM commands will appear here."
        className={`${baseClass}__empty-feed`}
      />
    );
  }
  return <div className={baseClass} />;
};

export default UpcomingCommandFeed;
