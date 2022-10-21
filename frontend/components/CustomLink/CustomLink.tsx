import React from "react";

import Icon from "components/Icon";

interface ICustomLinkProps {
  url: string;
  text: string;
  newTab?: boolean;
  //* Icon wraps on new line with last word */
  multiline?: boolean;
}

const baseClass = "custom-link";

const CustomLink = ({
  url,
  text,
  newTab = false,
  multiline = false,
}: ICustomLinkProps): JSX.Element => {
  const target = newTab ? "_blank" : "";

  const multilineText = text.substring(0, text.lastIndexOf(" ") + 1);
  const lastWord = text.substring(text.lastIndexOf(" ") + 1, text.length);

  const content = multiline ? (
    <>
      {multilineText}
      <span className={`${baseClass}__no-wrap`}>
        {lastWord}
        <Icon name="external-link" className={`${baseClass}__external-icon`} />
      </span>
    </>
  ) : (
    <>
      {text}
      <Icon name="external-link" className={`${baseClass}__external-icon`} />
    </>
  );

  return (
    <a
      href={url}
      target={target}
      rel="noopener noreferrer"
      className={baseClass}
    >
      {content}
    </a>
  );
};

export default CustomLink;
