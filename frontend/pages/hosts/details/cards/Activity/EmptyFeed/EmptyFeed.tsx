import React from "react";

import EmptyState from "components/EmptyState";

interface IEmptyFeedProps {
  title: string;
  message: string;
  className?: string;
}

const EmptyFeed = ({ title, message, className }: IEmptyFeedProps) => {
  return (
    <EmptyState
      variant="list"
      header={title}
      info={message}
      className={className}
    />
  );
};

export default EmptyFeed;
