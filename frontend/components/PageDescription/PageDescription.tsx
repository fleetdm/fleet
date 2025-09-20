import React from "react";
import classnames from "classnames";

const baseClass = "page-description";

interface IPageDescription {
  content: React.ReactNode;
  variant?: "default" | "card" | "tab-panel" | "right-panel";
}

const PageDescription = ({ content, variant }: IPageDescription) => {
  const classNames = classnames(baseClass, {
    [`${baseClass}__card`]: variant === "card",
    [`${baseClass}__tab-panel`]: variant === "tab-panel",
    [`${baseClass}__right-panel`]: variant === "right-panel",
  });

  return (
    <div className={classNames}>
      <p>{content}</p>
    </div>
  );
};

export default PageDescription;
