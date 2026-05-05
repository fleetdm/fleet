import React from "react";
import classnames from "classnames";

const baseClass = "page-description";

interface IPageDescription {
  content: React.ReactNode;
  /** Section descriptions styles differ from page level descriptions */
  variant?: "card" | "tab-panel" | "right-panel" | "modal";
  className?: string;
}

const sectionVariants = ["card", "tab-panel", "right-panel", "modal"];

const PageDescription = ({ content, variant, className }: IPageDescription) => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}__section-description`]:
      variant && sectionVariants.includes(variant),
  });

  return (
    <div className={classNames}>
      <p>{content}</p>
    </div>
  );
};

export default PageDescription;
