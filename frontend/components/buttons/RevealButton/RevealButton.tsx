import React from "react";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

export interface IRevealButtonProps {
  isShowing: boolean;
  baseClass: string;
  hideText: string;
  showText: string;
  caretBefore?: boolean;
  caretAfter?: boolean;
  autofocus?: boolean;
  disabled?: boolean;
  tooltipHtml?: string;
  onClick?:
    | ((value?: any) => void)
    | ((evt: React.MouseEvent<HTMLButtonElement>) => void);
}

const RevealButton = ({
  isShowing,
  hideText,
  showText,
  caretBefore,
  caretAfter,
  autofocus,
  disabled,
  tooltipHtml,
  onClick,
}: IRevealButtonProps): JSX.Element => {
  const classNameGenerator = () => {
    if (caretBefore) {
      return isShowing ? "reveal upcaretbefore" : "reveal rightcaretbefore";
    }
    if (caretAfter) {
      return isShowing ? "reveal upcaretafter" : "reveal downcaretafter";
    }
  };

  const buttonText = isShowing ? hideText : showText;

  return (
    <Button
      variant="unstyled"
      className={`reveal-button ${classNameGenerator()}`}
      onClick={onClick}
      autofocus={autofocus}
      disabled={disabled}
    >
      {tooltipHtml ? (
        <TooltipWrapper tipContent={tooltipHtml}>{buttonText}</TooltipWrapper>
      ) : (
        buttonText
      )}
    </Button>
  );
};

export default RevealButton;
