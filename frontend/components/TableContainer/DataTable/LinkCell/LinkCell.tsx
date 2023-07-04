import React from "react";

// using browserHistory directly because "router"
// is difficult to pass as a prop
import { browserHistory } from "react-router";

import Button from "components/buttons/Button/Button";

interface ILinkCellProps {
  value: string | JSX.Element;
  path: string;
  title?: string;
  classes?: string;
  customOnClick?: () => void;
}

const LinkCell = ({
  value,
  path,
  title,
  classes,
  customOnClick,
}: ILinkCellProps): JSX.Element => {
  const onClick = (): void => {
    customOnClick && customOnClick();
    browserHistory.push(path);
  };

  return (
    <Button
      className={`link-cell ${classes}`}
      onClick={onClick}
      variant="text-link"
      title={title}
    >
      {value}
    </Button>
  );
};

export default LinkCell;
