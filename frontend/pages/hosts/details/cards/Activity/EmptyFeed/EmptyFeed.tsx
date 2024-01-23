import React from "react";
import classnames from "classnames";

const baseClass = "empty-feed";

interface IEmptyFeedProps {
  title: string;
  message: string;
  className?: string;
}

const EmptyFeed = ({ title, message, className }: IEmptyFeedProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <p className={`${baseClass}__title`}>{title}</p>
      <p>{message}</p>
    </div>
  );
};

export default EmptyFeed;
