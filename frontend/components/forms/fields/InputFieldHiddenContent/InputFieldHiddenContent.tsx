import React, { useState } from "react";

import { stringToClipboard } from "utilities/copy_text";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import classnames from "classnames";

const baseClass = "input-field-hidden-content";

interface IInputFieldHiddenContentProps {
  value: string;
  name?: string;
  className?: string;
}

const InputFieldHiddenContent = ({
  value,
  name,
  className,
}: IInputFieldHiddenContentProps) => {
  const [copyMessage, setCopyMessage] = useState("");
  const [showSecret, setShowSecret] = useState(false);

  const classNames = classnames(baseClass, className);

  const onCopySecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(value)
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    setTimeout(() => setCopyMessage(""), 1000);

    return false;
  };

  const onToggleSecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    setShowSecret(!showSecret);
    return false;
  };

  const renderLabel = () => {
    return (
      <span className={`${baseClass}__name`}>
        <span className="buttons">
          {copyMessage && (
            <span
              className={`${baseClass}__copy-message`}
            >{`${copyMessage} `}</span>
          )}
          <Button
            variant="unstyled"
            className={`${baseClass}__copy-secret-icon`}
            onClick={onCopySecret}
          >
            <Icon name="copy" />
          </Button>
          <Button
            variant="unstyled"
            className={`${baseClass}__show-secret-icon`}
            onClick={onToggleSecret}
          >
            <Icon name="eye" />
          </Button>
        </span>
      </span>
    );
  };

  return (
    <div className={classNames}>
      <InputField
        disabled
        inputWrapperClass={`${baseClass}__secret-input`}
        name={name}
        label={renderLabel()}
        type={showSecret ? "text" : "password"}
        value={value}
      />
    </div>
  );
};

export default InputFieldHiddenContent;
