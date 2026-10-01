import React from "react";

import Button from "components/buttons/Button";
import { ButtonVariant } from "components/buttons/Button/Button";
// @ts-ignore
import DropdownButton from "components/buttons/DropdownButton";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon/Icon";
import { IconNames } from "components/icons";

// TODO - there are two `IActionButtonProps` in the codebase, one specifically used in
// TableContainer. Disambiguate these names or combine into a single abstraction.
export interface IActionButtonProps {
  type: "primary" | "secondary";
  label: string;
  onClick: () => void;
  buttonVariant?: ButtonVariant;
  iconName?: IconNames;
  hideAction?: boolean;
  gitOpsModeCompatible?: boolean;
  disabled?: boolean;
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
              <Button
                key={action.label}
                onClick={action.onClick}
                disabled={action.disabled}
              >
                {action.label}
              </Button>
            )
        )}
      </div>
      <div className={`${baseClass}__action-buttons--secondary`}>
        <div
          className={`${baseClass}__action-buttons--secondary-buttons action-buttons__secondary-buttons`}
        >
          {secondaryActions.map((action) => {
            const button = (
              <Button
                key={action.label}
                variant={action.buttonVariant}
                onClick={action.onClick}
                disabled={action.disabled}
                icon={action.iconName}
              >
                {action.label}
              </Button>
            );
            // A GitOps-compatible action is only disabled by GitOps mode, so explain it.
            if (action.gitOpsModeCompatible && action.disabled) {
              return (
                <GitOpsModeTooltipWrapper
                  key={action.label}
                  renderChildren={() => button}
                />
              );
            }
            return button;
          })}
        </div>
        <div
          className={`${baseClass}__action-buttons--secondary-dropdown action-buttons__secondary-dropdown`}
        >
          <DropdownButton
            showCaret={false}
            options={secondaryActions}
            variant="secondary"
          >
            More options <Icon name="more" />
          </DropdownButton>
        </div>
      </div>
    </div>
  );
};

export default ActionButtons;
