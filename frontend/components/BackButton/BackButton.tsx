import React from "react";
import { browserHistory } from "react-router";

import classnames from "classnames";
import Button from "components/buttons/Button";

interface IBackButtonProps {
  /** Default: "Back" */
  text?: string;
  path?: string;
  className?: string;
}

const baseClass = "back-button";

const BackButton = ({
  text = "Back",
  path,
  className,
}: IBackButtonProps): JSX.Element => {
  const classes = classnames(baseClass, className);

  const onClick = (): void => {
    if (path) {
      browserHistory.push(path);
    } else browserHistory.goBack();
  };

  return (
    <Button
      variant="subdued"
      onClick={onClick}
      className={classes}
      icon="chevron-left"
    >
      <span>{text}</span>
    </Button>
  );
};
export default BackButton;
