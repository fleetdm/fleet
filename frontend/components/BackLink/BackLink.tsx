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
  const backLinkClass = classnames(baseClass, className);

  const onClick = (): void => {
    if (path) {
      browserHistory.push(path);
    } else browserHistory.goBack();
  };

  return (
    /* Need to update react-router to use Link component to go back on FF */
    <Button onClick={onClick} className={backLinkClass} variant="text-link">
      <>
        <Icon
          name="chevron"
          className={`${baseClass}__back-icon`}
          direction="left"
          color="core-fleet-blue"
        />
        <span>{text}</span>
      </>
    </Button>
  );
};
export default BackLink;
