import React from 'react';

interface ITextCellProps {
  value: string | number;
  formatter?: (value: string | number) => string
}

const TextCell = (props: ITextCellProps): JSX.Element => {
  const {
    value,
    formatter = val => val, // identity function if no formatter is provided
  } = props;

  return (
    <span>
      {formatter(value)}
    </span>
  );
};

export default TextCell;
