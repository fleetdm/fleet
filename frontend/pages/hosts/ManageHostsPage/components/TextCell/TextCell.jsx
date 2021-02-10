import React from 'react';

const TextCell = (props) => {
  const { value } = props;

  return (
    <span>
      {value}
    </span>
  );
};

export default TextCell;
