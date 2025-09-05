import Button from "components/buttons/Button";
import Icon from "components/Icon";
import React from "react";

const baseClass = "info-button";

interface IInfoButton {
  toggleInfoModal: () => void;
}

const InfoButton = ({ toggleInfoModal }: IInfoButton) => {
  return (
    <Button
      className={baseClass}
      onClick={() => toggleInfoModal()}
      variant="text-icon"
    >
      <>
        Info <Icon name="info" size="small" />
      </>
    </Button>
  );
};

export default InfoButton;
