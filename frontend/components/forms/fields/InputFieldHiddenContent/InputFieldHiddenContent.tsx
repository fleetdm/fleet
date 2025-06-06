import React from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import classnames from "classnames";

const baseClass = "input-field-hidden-content";

interface IInputFieldHiddenContentProps {
  value: string;
  name?: string;
  className?: string;
  helpText?: string | JSX.Element;
}

/** Used to easily create an InputField with a show/hide and copy buttion */
const InputFieldHiddenContent = ({
  value,
  name,
  className,
  helpText,
}: IInputFieldHiddenContentProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <div className={classNames}>
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__secret-input`}
        name={name}
        enableShowSecret
        enableCopy
        type={"password"}
        value={value}
        helpText={helpText}
      />
    </div>
  );
};

export default InputFieldHiddenContent;
