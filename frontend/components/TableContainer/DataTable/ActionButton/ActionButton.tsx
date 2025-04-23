import React, { useCallback } from "react";
import { kebabCase, noop } from "lodash";
import classnames from "classnames";

import { ButtonVariant } from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";

import Button from "../../../buttons/Button";

const baseClass = "action-button";
export interface IActionButtonProps {
  name: string;
  buttonText: string | ((targetIds: number[]) => string);
  onClick?: (ids: number[]) => void;
  targetIds?: number[]; // TODO figure out undefined case
  variant?: ButtonVariant;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  iconSvg?: IconNames;
  iconStroke?: boolean;
  iconPosition?: string;
  isDisabled?: boolean;
  tooltipContent?: React.ReactNode;
}

function useActionCallback(
  callbackFn: (targetIds: number[]) => void | undefined
) {
  return useCallback(
    (targetIds: any) => {
      callbackFn(targetIds);
    },
    [callbackFn]
  );
}

const ActionButton = (buttonProps: IActionButtonProps): JSX.Element | null => {
  const {
    name,
    buttonText,
    onClick,
    targetIds = [],
    variant = "default",
    hideButton,
    iconSvg,
    iconStroke = false,
    iconPosition,
    isDisabled,
    tooltipContent,
  } = buttonProps;
  const onButtonClick = useActionCallback(onClick || noop);

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

  if (isHidden(hideButton)) {
    return null;
  }

  const buttonClasses = classnames(
    baseClass,
    `${baseClass}__${kebabCase(name)}`,
    { [`${baseClass}__disabled`]: isDisabled }
  );

  const renderButton = () => (
    <div className={buttonClasses}>
      <Button
        onClick={() => onButtonClick(targetIds)}
        variant={variant}
        iconStroke={iconStroke}
      >
        <>
          {iconPosition === "left" && iconSvg && <Icon name={iconSvg} />}
          {buttonText}
          {iconPosition !== "left" && iconSvg && <Icon name={iconSvg} />}
        </>
      </Button>
    </div>
  );

  if (tooltipContent) {
    return (
      <div className={baseClass}>
        <TooltipWrapper
          tipContent={tooltipContent}
          position="top"
          fixedPositionStrategy
          underline={false}
          clickable={false}
          showArrow
        >
          {renderButton()}
        </TooltipWrapper>
      </div>
    );
  }
  return renderButton();
};

export default ActionButton;
