import React from "react";

interface ISpinnerProps {
  isInButton?: boolean;
}

const Spinner = ({ 
  isInButton, 
}: ISpinnerProps): JSX.Element => {
  if (isInButton) {
    return (
      <div className="ring ring-for-button">
        <div />
        <div />
        <div />
        <div />
      </div>
    );
  }
  
  return (
    <div className="card">
      <div className="ring">
        <div />
        <div />
        <div />
        <div />
      </div>
    </div>
  );
};

export default Spinner;
