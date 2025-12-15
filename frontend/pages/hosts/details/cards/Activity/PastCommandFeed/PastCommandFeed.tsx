import React from "react";

import { ICommand } from "interfaces/command";

import EmptyFeed from "../EmptyFeed/EmptyFeed";

const baseClass = "past-command-feed";

interface IPastCommandFeedProps {
  commands: ICommand[];
}

const PastCommandFeed = ({ commands }: IPastCommandFeedProps) => {
  if (commands.length === 0) {
    return (
      <EmptyFeed
        title="No MDM commands"
        message="Completed MDM commands will appear here."
        className={`${baseClass}__empty-feed`}
      />
    );
  }

  return <div className={baseClass}>past commands</div>;
};

export default PastCommandFeed;
