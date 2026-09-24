import React from "react";

const baseClass = "library-item-accordion-list";

interface ILibraryItemAccordionListProps {
  children: React.ReactNode;
  className?: string;
}

const LibraryItemAccordionList = ({
  children,
  className,
}: ILibraryItemAccordionListProps) => {
  const classes = className ? `${baseClass} ${className}` : baseClass;
  return <div className={classes}>{children}</div>;
};

export default LibraryItemAccordionList;
