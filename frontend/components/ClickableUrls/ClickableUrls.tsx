import React from "react";
import * as DOMPurify from "dompurify";
import classnames from "classnames";

interface IClickableUrls {
  text: string;
  className?: string;
}

const baseClass = "back-link";

const ClickableUrls = ({ text, className }: IClickableUrls): JSX.Element => {
  const clickableUrlClasses = classnames(baseClass, className);

  // take that text, identify all links
  const findLinks = (): string => {
    return text;
  };

  const replacedLinks = text.replace(
    /(^|[^a-z0-9\.\-\/])(https?:\/\/[a-z0-9\.\-\/]+)([^a-z0-9\.\-\/]|$)/g,
    '$1<a href="$2" target="_blank">$2</a>$3'
  );

  const sanitizedResolutionContent = DOMPurify.sanitize(replacedLinks);

  const textWithLinks = (
    <div
      className={clickableUrlClasses}
      dangerouslySetInnerHTML={{ __html: sanitizedResolutionContent }}
    />
  );
  // take those links and replace them with custom links to a new tab
  // Return as JSX.Element

  return textWithLinks;
};
export default ClickableUrls;
