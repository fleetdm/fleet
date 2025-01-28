import React from "react";
import * as DOMPurify from "dompurify";
import classnames from "classnames";

interface IClickableUrls {
  text: string;
  className?: string;
}

const baseClass = "clickable-urls";

const urlReplacer = (match: string) => {
  const url = match.startsWith("http") ? match : `https://${match}`;
  return `<a href=${url} target="_blank" rel="noreferrer">
      ${match}
    </a>`;
};

const ClickableUrls = ({ text, className }: IClickableUrls): JSX.Element => {
  const clickableUrlClasses = classnames(baseClass, className);

  // Regex to find case insensitive URLs and replace with link
  const textWithLinks = text.replaceAll(
    /(((https?)?(:\/\/))|((https?)?(:\/\/)?(www\.)))[-a-zA-Z0-9@:%._+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_+.~#?&//=]*)/g,
    urlReplacer
  );
  const sanitizedTextWithLinks = DOMPurify.sanitize(textWithLinks, {
    ADD_ATTR: ["target"], // Allows opening in a new tab
  });

  return (
    <div
      className={clickableUrlClasses}
      dangerouslySetInnerHTML={{ __html: sanitizedTextWithLinks }}
    />
  );
};
export default ClickableUrls;
