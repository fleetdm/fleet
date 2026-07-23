import React from "react";
import classnames from "classnames";

import Button, { IButtonProps } from "components/buttons/Button";

const baseClass = "automations-button";

export type IAutomationsButtonProps = Pick<
  IButtonProps,
  "onClick" | "disabled" | "className" | "size"
>;

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
      variant="secondary"
      size={size}
    >
      Manage automations
    </Button>
  );
};

export default AutomationsButton;
