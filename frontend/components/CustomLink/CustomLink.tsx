import React from "react";

import Icon from "components/Icon";

interface ICustomLinkProps {
  url: string;
  text: string;
  /** Default opens in a new tab */
  newTab?: boolean;
  //* Icon wraps on new line with last word */
  multiline?: boolean;
}

const baseClass = "custom-link";

const CustomLink = ({
  url,
  text,
  newTab = true,
  multiline = false,
}: ICustomLinkProps): JSX.Element => {
  const target = newTab ? "_blank" : "";

  // External link icon never pushed to a line alone
  if (multiline) {
    const multilineText = text.substring(0, text.lastIndexOf(" ") + 1);
    const lastWord = text.substring(text.lastIndexOf(" ") + 1, text.length);

    return (
      <a
        href={url}
        target={target}
        rel="noopener noreferrer"
        className={baseClass}
      >
        {multilineText}
        <span className="no-wrap">
          {lastWord}
          <Icon name="external-link" />
        </span>
      </a>
    );
  }

  return (
    <a
      href={url}
      target={target}
      rel="noopener noreferrer"
      className={baseClass}
    >
      {text}
      <Icon name="external-link" />
    </a>
  );
};
export default CustomLink;
