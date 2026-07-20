import React, { useEffect, useRef, useState } from "react";
import classnames from "classnames";
import { Tooltip as ReactTooltip5 } from "react-tooltip-5";
import { uniqueId } from "lodash";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { stringToClipboard } from "utilities/copy_text";

type CopyButtonVariant =
  | "icon"
  | "inverse"
  | "secondary"
  | "subdued"
  | "compact";

interface ICopyButtonProps {
  copyText: string;
  /** Override the button's content. Defaults to `<Icon name="copy" />`. */
  children?: React.ReactNode;
  /** `"icon"` (default) — standard 36×36 icon button.
   *  `"inverse"` — bordered icon-with-text button (use with children).
   *  `"secondary"` — bordered 36×36 icon-only button (the current preferred
   *  secondary style — see #35329).
   *  `"subdued"` — borderless low-emphasis icon-only button — see #35329.
   *  `"compact"` — icon collapsed to its natural size, no extra vertical
   *  chrome. Use for inline-with-text copy actions so the surrounding row
   *  doesn't grow to button height. */
  variant?: CopyButtonVariant;
  size?: "small" | "default";
  className?: string;
  ariaLabel?: string;
  /** Distance in px from the anchor to the tooltip. Defaults to `4` — tight
   *  for inline-with-text copy actions. Pass `10` to match react-tooltip 5's
   *  own default when the trigger is a larger floating button. */
  tooltipOffset?: number;
  /** Table buttons show on row hover and tab focus only */
  rowHover?: boolean;
}

const baseClass = "copy-button";
const HIDE_AFTER_MS = 1000;
const SUCCESS_MESSAGE = "Copied!";
const ERROR_MESSAGE = "Copy failed";

const CopyButton = ({
  copyText,
  children,
  variant = "icon",
  size,
  className,
  ariaLabel = "Copy to clipboard",
  tooltipOffset = 4,
  rowHover = false,
}: ICopyButtonProps) => {
  const [message, setMessage] = useState<string | null>(null);
  const tipIdRef = useRef(uniqueId("copy-button-tooltip-"));
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const onClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    // Fleet Button's --icon variant uses :focus (not :focus-visible) for its
    // hover-background, so a mouse click leaves the button visually "stuck"
    // highlighted. Drop focus only for mouse activations — keyboard Enter/
    // Space report `detail === 0` and must keep their tab position.
    if (evt.detail !== 0) {
      evt.currentTarget.blur();
    }

    // Cancel any previous click's hide timer so the new badge gets the full
    // window — and so the hide doesn't race a slow `writeText()` (>1s would
    // leave the message visible with no timer to clear it).
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }

    const scheduleHide = () => {
      timerRef.current = setTimeout(() => setMessage(null), HIDE_AFTER_MS);
    };

    stringToClipboard(copyText)
      .then(() => {
        setMessage(SUCCESS_MESSAGE);
        scheduleHide();
      })
      .catch(() => {
        setMessage(ERROR_MESSAGE);
        scheduleHide();
      });
  };

  const isCompact = variant === "compact";

  return (
    <span className={baseClass} data-tooltip-id={tipIdRef.current}>
      <Button
        variant={isCompact ? "icon" : variant}
        size={size}
        iconStroke
        onClick={onClick}
        className={classnames(
          `${baseClass}__button`,
          {
            [`${baseClass}__button--compact`]: isCompact,
            "row-hover-button": rowHover,
          },
          className
        )}
        ariaLabel={ariaLabel}
      >
        {children ?? (
          <Icon name="copy" size={size === "small" ? "small" : undefined} />
        )}
      </Button>
      <ReactTooltip5
        id={tipIdRef.current}
        isOpen={message !== null}
        place="left"
        offset={tooltipOffset}
        opacity={1}
        disableStyleInjection
        noArrow
        className={`${baseClass}__tooltip`}
      >
        {message}
      </ReactTooltip5>
    </span>
  );
};

export default CopyButton;
