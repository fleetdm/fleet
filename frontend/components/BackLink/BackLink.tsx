// This is renamed to BackButton in larger PR, needed to update component to Button to properly style
// as I removed direction-link mixin in larger PR.

import React from "react";
import { browserHistory } from "react-router";

import Button from "components/buttons/Button";
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
    <Button
      className={classnames(baseClass, className)}
      onClick={onClick}
      variant="inverse"
    >
      <>
        <Icon name="chevron-left" color="core-fleet-blue" />
        <span>{text}</span>
      </>
    </Button>
  );
};
export default BackLink;
