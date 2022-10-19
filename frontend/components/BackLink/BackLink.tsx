import React from "react";
import { browserHistory } from "react-router";
import Icon from "components/Icon";
import Button from "components/buttons/Button";

interface IBackLinkProps {
  text: string;
  path?: string;
}

const BackLink = ({ text, path }: IBackLinkProps): JSX.Element => {
  const onClick = (): void => {
    if (path) {
      browserHistory.push(path);
    } else browserHistory.goBack();
  };

  return (
    <Button className={"back-link"} onClick={onClick} variant="text-icon">
      <>
        <Icon name={"chevron-left"} />
        {text}
      </>
    </Button>
  );
};
export default BackLink;
