import React, { useCallback } from "react";
import { kebabCase } from "lodash";

import { ButtonVariant } from "components/buttons/Button/Button";
import Button from "components/buttons/Button";
import { IconNames } from "components/icons";
import Icon from "components/Icon/Icon";

import CloseIcon from "../../../../../assets/images/icon-close-vibrant-blue-16x16@2x.png";
import DeleteIcon from "../../../../../assets/images/icon-delete-vibrant-blue-12x14@2x.png";
import CheckIcon from "../../../../../assets/images/icon-action-check-16x15@2x.png";
import DisableIcon from "../../../../../assets/images/icon-action-disable-14x14@2x.png";
import TransferIcon from "../../../../../assets/images/icon-action-transfer-16x16@2x.png";

const baseClass = "action-button";
export interface IActionButtonProps {
  name: string;
  buttonText: string;
  onActionButtonClick: (ids: number[]) => void | undefined;
  targetIds?: number[]; // TODO figure out undefined case
  variant?: ButtonVariant;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  iconName?: IconNames;
  iconPosition?: string;
}

function useActionCallback(
  callbackFn: (targetIds: number[]) => void | undefined
) {
  return useCallback(
    (targetIds) => {
      callbackFn(targetIds);
    },
    [callbackFn]
  );
}

const ActionButton = (buttonProps: IActionButtonProps): JSX.Element | null => {
  const {
    name,
    buttonText,
    onActionButtonClick,
    targetIds = [],
    variant = "brand",
    hideButton,
    iconName,
    iconPosition,
  } = buttonProps;
  const onButtonClick = useActionCallback(onActionButtonClick);

  // hideButton is intended to provide a flexible way to specify show/hide conditions via a boolean or a function that evaluates to a boolean
  // currently it is typed to accept an array of targetIds but this typing could easily be expanded to include other use cases
  const isHidden = (
    hideButtonProp: boolean | ((ids: number[]) => boolean) | undefined
  ) => {
    if (typeof hideButtonProp === "function") {
      return hideButtonProp(targetIds);
    }
    return Boolean(hideButtonProp);
  };

  return isHidden(hideButton) ? null : (
    <div className={`${baseClass} ${baseClass}__${kebabCase(name)}`}>
      <Button onClick={() => onButtonClick(targetIds)} variant={variant}>
        <>
          {iconPosition === "left" && iconName && <Icon name={iconName} />}
          {buttonText}
          {iconPosition !== "left" && iconName && <Icon name={iconName} />}
        </>
      </Button>
    </div>
  );
};

export default ActionButton;
