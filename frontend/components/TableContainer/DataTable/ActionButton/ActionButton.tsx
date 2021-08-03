import React, { useCallback } from "react";
import PropTypes from "prop-types";
import { kebabCase } from "lodash";

// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Button from "../../../buttons/Button";
import CloseIcon from "../../../../../assets/images/icon-close-vibrant-blue-16x16@2x.png";
import DeleteIcon from "../../../../../assets/images/icon-delete-vibrant-blue-12x14@2x.png";

const baseClass = "action-button";
export interface IActionButtonProps {
  name: string;
  buttonText: string;
  onActionButtonClick: (targetIds: number[]) => void | undefined;
  targetIds?: number[]; // TODO figure out undefined case
  variant?: string;
  hideButton?: boolean | ((targetIds: number[]) => boolean);
  icon?: string;
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

const ActionButton = (props: IActionButtonProps): JSX.Element | null => {
  const {
    name,
    buttonText,
    onActionButtonClick,
    targetIds = [],
    variant,
    hideButton,
    icon,
    iconPosition,
  } = props;
  const onButtonClick = useActionCallback(onActionButtonClick);

  const iconLink = ((iconProp) => {
    // check if using pre-defined short-hand otherwise otherwise return the prop
    switch (iconProp) {
      case "close":
        return CloseIcon;
      case "remove":
        return CloseIcon;
      case "delete":
        return DeleteIcon;
      default:
        return null;
    }
  })(icon);
  // hideButton is intended to provide a flexible way to specify show/hide conditions via a boolean or a function that evaluates to a boolean
  // currently it is typed to accept an array of targetIds but this typing could easily be expanded to include other use cases
  const testCondition = (
    hideButtonProp: boolean | ((targetIds: number[]) => boolean) | undefined
  ) => {
    if (typeof hideButtonProp === "function") {
      return hideButtonProp;
    }
    return () => Boolean(hideButtonProp);
  };
  const isHidden = testCondition(hideButton)(targetIds);

  return !isHidden ? (
    <div className={`${baseClass} ${baseClass}__${kebabCase(name)}`}>
      <Button onClick={() => onButtonClick(targetIds)} variant={variant}>
        <>
          {iconPosition === "left" && iconLink && (
            <img alt={`${name} icon`} src={iconLink} />
          )}
          {buttonText}
          {iconPosition !== "left" && iconLink && (
            <img alt={`${name} icon`} src={iconLink} />
          )}
        </>
      </Button>
    </div>
  ) : null;
};

ActionButton.propTypes = {
  name: PropTypes.string,
  buttonText: PropTypes.string,
  onActionButtonClick: PropTypes.func,
  targetIds: PropTypes.arrayOf(PropTypes.number),
  variant: PropTypes.string,
  hideButton: PropTypes.oneOfType([PropTypes.bool, PropTypes.func]),
  icon: PropTypes.string,
  iconPosition: PropTypes.string,
};

export default ActionButton;
