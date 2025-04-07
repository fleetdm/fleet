import React from "react";
import classnames from "classnames";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

export interface IRevealButtonProps {
  isShowing: boolean;
  className?: string;
  hideText: string;
  showText: string;
  caretPosition?: "before" | "after";
  autofocus?: boolean;
  disabled?: boolean;
  tooltipContent?: React.ReactNode;
  disabledTooltipContent?: React.ReactNode;
  onClick?:
    | ((value?: any) => void)
    | ((evt: React.MouseEvent<HTMLButtonElement>) => void);
}

const baseClass = "reveal-button";

const RevealButton = ({
  isShowing,
  className,
  hideText,
  showText,
  caretPosition,
  autofocus,
  disabled,
  tooltipContent,
  disabledTooltipContent,
  onClick,
}: IRevealButtonProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  const buttonContent = () => {
    const text = isShowing ? hideText : showText;

    const buttonText =
      tooltipContent && !disabled ? (
        <TooltipWrapper tipContent={tooltipContent}>{text}</TooltipWrapper>
      ) : (
        text
      );

    return (
      <>
        {caretPosition === "before" && (
          <Icon
            name={isShowing ? "chevron-down" : "chevron-right"}
            color="core-fleet-blue"
          />
        )}
        {buttonText}
        {caretPosition === "after" && (
          <Icon
            name={isShowing ? "chevron-up" : "chevron-down"}
            color="core-fleet-blue"
          />
        )}
      </>
    );
  };

  const button = (
    <Button
      variant="text-icon"
      className={classNames}
      onClick={onClick}
      autofocus={autofocus}
      disabled={disabled}
    >
      {buttonContent()}
    </Button>
  );

  if (disabled && disabledTooltipContent) {
    // wrap the tooltip around the Button so it works while disabled
    return (
      <TooltipWrapper
        tipContent={disabledTooltipContent}
        showArrow
        underline={false}
        position="right"
        tipOffset={12}
      >
        {button}
      </TooltipWrapper>
    );
  }
  return button;
};

export default RevealButton;
