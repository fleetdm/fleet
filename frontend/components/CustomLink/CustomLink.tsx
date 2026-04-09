import React from "react";

import Icon from "components/Icon";
import classnames from "classnames";
import { Colors } from "styles/var/colors";

interface ICustomLinkProps {
  url?: string;
  text: string;
  className?: string;
  /** open the link in a new tab
   * @default false
   */
  newTab?: boolean;
  /** Icon wraps on new line with last word */
  multiline?: boolean;
  /** Restricts access via keyboard when CustomLink is part of disabled UI */
  disableKeyboardNavigation?: boolean;
  /**
   * Changes the appearance of the link.
   *
   * @default "default"
   */
  variant?: "tooltip-link" | "banner-link" | "flash-message-link" | "default";
  customClickHandler?: (e: React.MouseEvent) => void;
}

const baseClass = "custom-link";

const CustomLink = ({
  url,
  text,
  className,
  newTab = false,
  multiline = false,
  disableKeyboardNavigation = false,
  variant = "default",
  customClickHandler,
}: ICustomLinkProps): JSX.Element => {
  const getIconColor = (): Colors => {
    switch (variant) {
      case "tooltip-link":
      case "flash-message-link":
        return "core-fleet-white";
      case "banner-link":
        return "core-fleet-black";
      default:
        return "ui-fleet-black-75";
    }
  };

  const customLinkClass = classnames(baseClass, className, {
    [`${baseClass}--${variant}`]: variant !== "default",
    [`${baseClass}--multiline`]: multiline,
  });

  // Needed to not trigger clickable parent elements
  // e.g. cell/row handlers with a tooltip that has a custom link inside
  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();

    // prevent navigation when we’re handling it ourselves
    // e.g. designed underline links opening modals
    if (customClickHandler) {
      e.preventDefault();
      customClickHandler(e);
    }
  };

  // Handle "Enter" key presses for accessibility when a custom click handler is provided
  const handleKeyDown = (e: React.KeyboardEvent<HTMLAnchorElement>) => {
    if (!customClickHandler) {
      return;
    }

    if (e.key === "Enter") {
      e.preventDefault();
      e.stopPropagation();
      // Reuse the same logic as click
      customClickHandler((e as unknown) as React.MouseEvent<HTMLAnchorElement>);
    }
  };

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
      onClick={handleClick}
      onKeyDown={handleKeyDown}
    >
      {content}
    </a>
  );
};

export default CustomLink;
