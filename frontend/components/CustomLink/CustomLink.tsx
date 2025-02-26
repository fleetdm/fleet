import React from "react";

import Icon from "components/Icon";
import classnames from "classnames";
import { Colors } from "styles/var/colors";

interface ICustomLinkProps {
  url: string;
  text: string;
  className?: string;
  /** open the link in a new tab
   * @default false
   */
  newTab?: boolean;
  /** Icon wraps on new line with last word */
  multiline?: boolean;
  // TODO: Refactor to use variant
  iconColor?: Colors;
  // TODO: Refactor to use variant
  color?: "core-fleet-blue" | "core-fleet-black" | "core-fleet-white";
  /** Restricts access via keyboard when CustomLink is part of disabled UI */
  disableKeyboardNavigation?: boolean;
  /**
   * Changes the appearance of the link.
   *
   * @default "default"
   *
   * TODO:
   * Longterm: refactor 14 instances away from iconColor/color combo, which
   * usually are identical and repetitive, toward variants e.g. "banner-link"
   */
  variant?: "tooltip-link" | "default" | "flash-message-link";
}

const baseClass = "custom-link";

const CustomLink = ({
  url,
  text,
  className,
  newTab = false,
  multiline = false,
  iconColor = "core-fleet-blue",
  color = "core-fleet-blue",
  disableKeyboardNavigation = false,
  variant = "default",
}: ICustomLinkProps): JSX.Element => {
  const getIconColor = (): Colors => {
    switch (variant) {
      case "tooltip-link":
      case "flash-message-link":
        return "core-fleet-white";
      default:
        return iconColor;
    }
  };

  const customLinkClass = classnames(baseClass, className, {
    [`${baseClass}--black`]: color === "core-fleet-black",
    [`${baseClass}--white`]: color === "core-fleet-white",
    [`${baseClass}--${variant}`]: variant !== "default",
  });

  const target = newTab ? "_blank" : "";

  const multilineText = text.substring(0, text.lastIndexOf(" ") + 1);
  const lastWord = text.substring(text.lastIndexOf(" ") + 1, text.length);

  const content = multiline ? (
    <>
      {multilineText}
      <span className={`${baseClass}__no-wrap`}>
        {lastWord}
        {newTab && (
          <Icon
            name="external-link"
            className={`${baseClass}__external-icon`}
            color={getIconColor()}
          />
        )}
      </span>
    </>
  ) : (
    <>
      {text}
      {newTab && (
        <Icon
          name="external-link"
          className={`${baseClass}__external-icon`}
          color={getIconColor()}
        />
      )}
    </>
  );

  return (
    <a
      href={url}
      target={target}
      rel="noopener noreferrer"
      className={customLinkClass}
      tabIndex={disableKeyboardNavigation ? -1 : 0}
    >
      {content}
    </a>
  );
};

export default CustomLink;
