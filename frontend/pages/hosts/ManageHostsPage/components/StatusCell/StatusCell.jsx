import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

const StatusCell = (props) => {
  const { value } = props;
  const statusClassName = classnames(
    'hosts-table__status',
    `hosts-table__status--${value}`,
  );

  return (
    <span className={statusClassName}>
      {value}
    </span>
  );
};

StatusCell.propTypes = {
  value: PropTypes.string,
};

export default StatusCell;
