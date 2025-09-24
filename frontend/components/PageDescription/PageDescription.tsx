import React from "react";

const baseClass = "page-description";

interface IPageDescription {
  content: React.ReactNode;
}

const PageDescription = ({ content }: IPageDescription) => {
  return (
    <div className={`${baseClass}`}>
      <p>{content}</p>
    </div>
  );
};

export default PageDescription;
