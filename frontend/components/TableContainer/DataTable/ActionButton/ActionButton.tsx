import React, { useCallback } from "react";
import { kebabCase, noop } from "lodash";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";

import { ButtonVariant } from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";
import { IconNames } from "components/icons";
import Button from "../../../buttons/Button";

const baseClass = "action-button";
export interface IActionButtonProps {
  name: string;
  buttonText: string | ((targetIds: number[]) => string);
  onActionButtonClick?: (ids: number[]) => void;
  targetIds?: number[]; // TODO figure out undefined case
  variant?: ButtonVariant;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  iconSvg?: IconNames;
  iconPosition?: string;
  indicatePremiumFeature?: boolean;
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
    onActionButtonClick,
    targetIds = [],
    variant = "brand",
    hideButton,
    iconSvg,
    iconPosition,
    indicatePremiumFeature,
  } = buttonProps;
  const onButtonClick = useActionCallback(onActionButtonClick || noop);

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

  return (
    <div className={`${baseClass} ${baseClass}__${kebabCase(name)}`}>
      {indicatePremiumFeature && (
        <PremiumFeatureIconWithTooltip tooltipDelayHide={500} />
      )}
      <Button
        disabled={indicatePremiumFeature}
        onClick={() => onButtonClick(targetIds)}
        variant={variant}
      >
        <>
          {iconPosition === "left" && iconSvg && <Icon name={iconSvg} />}
          {buttonText}
          {iconPosition !== "left" && iconSvg && <Icon name={iconSvg} />}
        </>
      </Button>
    </div>
  );
};

export default ActionButton;
