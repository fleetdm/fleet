import React from "react";

const baseClass = "empty-feed";

interface IEmptyFeedProps {
  title: string;
  message: string;
}

const EmptyFeed = ({ title, message }: IEmptyFeedProps) => {
  return (
    <div className={baseClass}>
      <p className={`${baseClass}__title`}>{title}</p>
      <p>{message}</p>
    </div>
  );
};

export default EmptyFeed;
