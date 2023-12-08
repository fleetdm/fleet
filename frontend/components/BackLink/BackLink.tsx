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
  const backLinkClass = classnames(baseClass, className);

  const onClick = (): void => {
    if (path) {
      browserHistory.push(path);
    } else browserHistory.goBack();
  };

  return (
    <Link className={backLinkClass} to={path || ""} onClick={onClick}>
      <>
        <Icon
          name="chevron-left"
          className={`${baseClass}__back-icon`}
          color="core-fleet-blue"
        />
        <span>{text}</span>
      </>
    </Link>
  );
};
export default BackLink;
