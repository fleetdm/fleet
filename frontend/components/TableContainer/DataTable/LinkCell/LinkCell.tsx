import React from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

import Button from "components/buttons/Button/Button";

interface ILinkCellProps {
  value: string;
  path: string;
  title?: string;
  classes?: string;
}

const LinkCell = ({
  value,
  path,
  title,
  classes = "w250",
}: ILinkCellProps): JSX.Element => {
  const dispatch = useDispatch();

  const onClick = (): void => {
    dispatch(push(path));
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
