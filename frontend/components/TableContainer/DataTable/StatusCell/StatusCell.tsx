import React from 'react';
import classnames from 'classnames';

interface IStatusCellProps {
  value: string;
}

const StatusCell = (props: IStatusCellProps): JSX.Element => {
  const { value } = props;

  const generateClassTag = (rawValue: string): string => {
    const classTag = rawValue.replace(' ', '-').toLowerCase();

    return classTag;
  };

  const statusClassName = classnames(
    'data-table__status',
    `data-table__status--${generateClassTag(value)}`,
  );

  return (
    <span className={statusClassName}>
      {value}
    </span>
  );
};

export default StatusCell;
