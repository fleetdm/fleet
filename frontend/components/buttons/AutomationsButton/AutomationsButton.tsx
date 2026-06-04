import React from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "automations-button";

export interface IAutomationsButtonProps {
  onClick?: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  disabled?: boolean;
  className?: string;
  size?: "small" | "wide" | "default";
}

const AutomationsButton = ({
  onClick,
  disabled,
  className,
  size,
}: IAutomationsButtonProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  return (
    <Button
      className={classNames}
      onClick={onClick}
      disabled={disabled}
      variant="inverse"
      size={size}
    >
      <Icon name="settings" /> Automations
    </Button>
  );
};

export default AutomationsButton;
