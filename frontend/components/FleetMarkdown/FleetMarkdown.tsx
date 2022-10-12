import classnames from "classnames";
import React from "react";
import ReactMarkdown from "react-markdown";

import ExternalLinkIcon from "../../../assets/images/icon-external-link-12x12@2x.png";

interface ICustomLinkProps {
  text: React.ReactNode;
  href: string;
  newTab?: boolean;
}

const CustomLink = ({ text, href, newTab = false }: ICustomLinkProps) => {
  const target = newTab ? "__blank" : "";
  return (
    <a href={href} target={target} rel="noopener noreferrer">
      {text}
      <img src={ExternalLinkIcon} alt="Open external link" />
    </a>
  );
};

interface IFleetMarkdownProps {
  markdown: string;
  className?: string;
}

const baseClass = "fleet-markdown";

/** This will give us sensible defaults for how we render markdown across the fleet application.
 * NOTE: can be extended later to take custom components, but dont need that at the moment.
 */
const FleetMarkdown = ({ markdown, className }: IFleetMarkdownProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <ReactMarkdown
      className={classNames}
      transformLinkUri={false}
      components={{
        a: ({ href = "", children }) => {
          return <CustomLink text={children} href={href} newTab />;
        },
      }}
    >
      {markdown}
    </ReactMarkdown>
  );
};

export default FleetMarkdown;
