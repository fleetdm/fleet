import React from "react";

import Button from "components/buttons/Button";

const baseClass = "info-button";

interface IInfoButton {
  onClick: () => void;
}

const InfoButton = ({ onClick }: IInfoButton) => {
  return (
    <Button
      className={baseClass}
      onClick={onClick}
      variant="subdued"
      icon="info"
      iconPosition="right"
    >
      Info
    </Button>
  );
};

export default InfoButton;
