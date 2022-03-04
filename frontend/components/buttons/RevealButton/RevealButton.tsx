import React from "react";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

export interface IRevealButtonProps {
  showBoolean: boolean;
  baseClass: string;
  hideString: string;
  showString: string;
  caratBefore?: boolean;
  caratAfter?: boolean;
  autofocus?: boolean;
  disabled?: boolean;
  tooltipHtml?: string;
  onClick?:
    | ((value?: any) => void)
    | ((evt: React.MouseEvent<HTMLButtonElement>) => void);
}

const RevealButton = ({
  showBoolean,
  baseClass,
  hideString,
  showString,
  caratBefore,
  caratAfter,
  autofocus,
  disabled,
  tooltipHtml,
  onClick,
}: IRevealButtonProps): JSX.Element => {
  const classNameGenerator = () => {
    if (caratBefore) {
      return showBoolean ? "reveal upcaratbefore" : "reveal rightcaratbefore";
    }
    if (caratAfter) {
      return showBoolean ? "reveal upcaratafter" : "reveal downcaratafter";
    }
  };

  if (tooltipHtml) {
    return (
      <Button
        variant="unstyled"
        className={`reveal-button ${classNameGenerator()}`}
        onClick={onClick}
        autofocus={autofocus}
        disabled={disabled}
      >
        <TooltipWrapper tipContent={tooltipHtml}>
          {showBoolean ? hideString : showString}
        </TooltipWrapper>
      </Button>
    );
  }

  return (
    <Button
      variant="unstyled"
      className={`${classNameGenerator()}
                    ${baseClass}__reveal-button`}
      onClick={onClick}
      autofocus={autofocus}
      disabled={disabled}
    >
      {showBoolean ? hideString : showString}
    </Button>
  );
};

export default RevealButton;
