import Button from "components/buttons/Button";
import Icon from "components/Icon";
import React from "react";

const baseClass = "info-button";

interface IInfoButton {
  onClick: () => void;
}

const InfoButton = ({ onClick }: IInfoButton) => {
  return (
    <Button className={baseClass} onClick={onClick} variant="inverse">
      <>
        Info <Icon name="info" size="small" />
      </>
    </Button>
  );
};

export default InfoButton;
