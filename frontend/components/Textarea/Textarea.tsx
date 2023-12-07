import React from "react";
import classnames from "classnames";

const baseClass = "textarea";

interface ITextareaProps {
  children: React.ReactNode;
  className?: string;
}

// A textarea component that encapsulates common styles and functionality.
const Textarea = ({ children, className }: ITextareaProps) => {
  const classNames = classnames(baseClass, className);
  return <div className={classNames}>{children}</div>;
};

export default Textarea;
