import React from "react";
import classnames from "classnames";

const baseClass = "textarea";

interface ITextareaProps {
  children: React.ReactNode;
  className?: string;
  label?: React.ReactNode;
  /** code - code font, max height 300px */
  variant?: "code" | "default";
}

// A textarea component that encapsulates common styles and functionality.
const Textarea = ({ children, className, label, variant }: ITextareaProps) => {
  // this is to preserve line breaks when we encounter a carriage return character.
  // We could not find a way to preserve line breaks in the CSS alone for this
  // character.
  if (typeof children === "string") {
    children = children.replace(/\r/g, "\n");
  }

  const wrapperClasses = classnames(`${baseClass}-wrapper`, className);
  const textareaClasses = classnames(baseClass, {
    [`${baseClass}--code`]: variant === "code",
  });

  return (
    <div className={wrapperClasses}>
      {label && <div className={`${baseClass}__label`}>{label}</div>}
      <div className={textareaClasses}>{children}</div>
    </div>
  );
};

export default Textarea;
