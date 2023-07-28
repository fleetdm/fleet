import React from "react";
import * as DOMPurify from "dompurify";
import classnames from "classnames";

interface IClickableUrls {
  text: string;
  className?: string;
}

const baseClass = "clickable-urls";

const ClickableUrls = ({ text, className }: IClickableUrls): JSX.Element => {
  const clickableUrlClasses = classnames(baseClass, className);

  // Regex to find case insensitive URLs and replace with link
  const replacedLinks = text.replaceAll(
    /(^|[^a-z0-9.\-/])(https?:\/\/[A-Za-z0-9.\-/]+)([^A-Za-z0-9.\-/]|$)/g,
    '$1<a href="$2" target="_blank">$2</a>$3'
  );

  const sanitizedResolutionContent = DOMPurify.sanitize(replacedLinks, {
    ADD_ATTR: ["target"], // Allows opening in a new tab
  });

  const textWithLinks = (
    <div
      className={clickableUrlClasses}
      dangerouslySetInnerHTML={{ __html: sanitizedResolutionContent }}
    />
  );

  return textWithLinks;
};
export default ClickableUrls;
