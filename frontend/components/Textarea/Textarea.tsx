import React from "react";
import classnames from "classnames";

const baseClass = "textarea";

interface ITextareaProps {
  children: React.ReactNode;
  className?: string;
}

// A textarea component that encapsulates common styles and functionality.
const Textarea = ({ children, className }: ITextareaProps) => {
  // this is to preserve line breaks when we encounter a carriage return character.
  // We could not find a way to preserve line breaks in the CSS alone for this
  // character.
  if (typeof children === "string") {
    children = children.replace(/\r/g, "\n");
  }

  const classNames = classnames(baseClass, className);
  return <div className={classNames}>{children}</div>;
};

export default Textarea;
