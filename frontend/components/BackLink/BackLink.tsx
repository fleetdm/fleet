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
    <Link to={path || ".."} onClick={onClick} className={backLinkClass}>
      <>
        <Icon
          name="chevron"
          className={`${baseClass}__back-icon`}
          direction="left"
          color="coreVibrantBlue"
        />
        {text}
      </>
    </Link>
  );
};
export default BackLink;
