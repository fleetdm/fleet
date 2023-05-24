import React, { useCallback } from "react";
import { kebabCase, noop } from "lodash";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";

import { ButtonVariant } from "components/buttons/Button/Button";
import Icon from "components/Icon/Icon";
import { IconNames } from "components/icons";
import Button from "../../../buttons/Button";
import CloseIcon from "../../../../../assets/images/icon-close-vibrant-blue-16x16@2x.png";
import CheckIcon from "../../../../../assets/images/icon-action-check-16x15@2x.png";
import DisableIcon from "../../../../../assets/images/icon-action-disable-14x14@2x.png";

const baseClass = "action-button";
export interface IActionButtonProps {
  name: string;
  buttonText: string | ((targetIds: number[]) => string);
  onActionButtonClick?: (ids: number[]) => void;
  targetIds?: number[]; // TODO figure out undefined case
  variant?: ButtonVariant;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  icon?: string;
  iconSvg?: IconNames;
  iconPosition?: string;
  indicatePremiumFeature?: boolean;
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
    icon,
    iconSvg,
    iconPosition,
    indicatePremiumFeature,
  } = buttonProps;
  const onButtonClick = useActionCallback(onActionButtonClick || noop);

  const iconLink = ((iconProp) => {
    // check if using pre-defined short-hand otherwise otherwise return the prop
    switch (iconProp) {
      case "close":
        return CloseIcon;
      case "remove":
        return CloseIcon;
      case "check":
        return CheckIcon;
      case "disable":
        return DisableIcon;
      default:
        return null;
    }
  })(icon);
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

  console.log("iconSvg", iconSvg);
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
          {iconPosition === "left" && iconLink && (
            <img alt={`${name} icon`} src={iconLink} />
          )}
          {iconPosition === "left" && iconSvg && <Icon name={iconSvg} />}
          {buttonText}
          {iconPosition !== "left" && iconLink && (
            <img alt={`${name} icon`} src={iconLink} />
          )}
          {iconPosition !== "left" && iconSvg && <Icon name={iconSvg} />}
        </>
      </Button>
    </div>
  );
};

export default ActionButton;
