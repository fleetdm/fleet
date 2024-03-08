import React from "react";
import { browserHistory, Link } from "react-router";

import Icon from "components/Icon";
import classnames from "classnames";

interface IBackLinkProps {
  text: string;
  path?: string;
  className?: string;
}

const baseClass = "back-link";

const BackLink = ({ text, path, className }: IBackLinkProps): JSX.Element => {
  const onClick = (): void => {
    if (path) {
      browserHistory.push(path);
    } else browserHistory.goBack();
  };

  return (
    <Link
      className={classnames(baseClass, className)}
      to={path || ""}
      onClick={onClick}
    >
      <>
        <Icon name="chevron-left" color="core-fleet-blue" />
        <span>{text}</span>
      </>
    </Link>
  );
};
export default BackLink;
