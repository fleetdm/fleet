import React from 'react';

const DropdownOption = (option) => {
  return (
    <span>
      <i className="kolidecon-add-button Select-icon" /> {option.label}
    </span>
  );
};

export default DropdownOption;
