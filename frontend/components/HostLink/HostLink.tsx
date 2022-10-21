import React from "react";
import { browserHistory } from "react-router";
import Icon from "components/Icon";
import Button from "components/buttons/Button";

interface IBackLinkProps {
  text: string;
  path?: string; // TODO
}

const baseClass = "host-link";

const HostLink = ({ text, path }: IBackLinkProps): JSX.Element => {
  const onClick = (): void => {
    if (path) {
      // TODO: Build to use build string from query params passed through
  };

  return (
    <Button className={baseClass} onClick={onClick} variant="text-icon">
      <>
        View all hosts
        <Icon name={"chevron-right"} className={`${baseClass}__link-icon`} />
      </>
    </Button>
  );
};
export default HostLink;
