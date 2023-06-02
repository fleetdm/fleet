import React from "react";

import Button from "components/buttons/Button";
import { ButtonVariant } from "components/buttons/Button/Button";
// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import Icon from "components/Icon/Icon";
import { IconNames } from "components/icons";

export interface IActionButtonProps {
  type: "primary" | "secondary";
  label: string;
  buttonVariant?: ButtonVariant;
  icon?: string;
  iconSvg?: IconNames;
  hideAction?: boolean;
  onClick: () => void;
}

interface IProps {
  baseClass: string;
  actions: IActionButtonProps[];
}

const ActionButtons = ({ baseClass, actions }: IProps): JSX.Element => {
  const primaryActions: IActionButtonProps[] = [];
  const secondaryActions: IActionButtonProps[] = [];

  actions.forEach((action) => {
    const { type, hideAction } = action;
    if (hideAction) {
      return;
    }
    type === "primary"
      ? primaryActions.push(action)
      : secondaryActions.push(action);
  });

  return (
    <div
      className={`${baseClass}__action-buttons action-buttons action-buttons__wrapper`}
    >
      <div className={`${baseClass}__action-buttons--primary`}>
        {primaryActions.map(
          (action) =>
            !action.hideAction && (
              <Button onClick={action.onClick}>{action.label}</Button>
            )
        )}
      </div>
      <div className={`${baseClass}__action-buttons--secondary`}>
        <div
          className={`${baseClass}__action-buttons--secondary-buttons action-buttons__secondary-buttons`}
        >
          {secondaryActions.map(
            (action) =>
              !action.hideAction &&
              (action.buttonVariant !== "text-icon" ? (
                <Button variant={action.buttonVariant} onClick={action.onClick}>
                  {action.label}
                </Button>
              ) : (
                <Button variant="text-icon" onClick={action.onClick}>
                  <>
                    {action.label}
                    {action.iconSvg && <Icon name={action.iconSvg} />}
                  </>
                </Button>
              ))
          )}
        </div>
        <div
          className={`${baseClass}__action-buttons--secondary-dropdown action-buttons__secondary-dropdown`}
        >
          <DropdownButton
            showCaret={false}
            options={secondaryActions}
            variant="text-icon"
          >
            More options <Icon name="more" />
          </DropdownButton>
        </div>
      </div>
    </div>
  );
};

export default ActionButtons;
