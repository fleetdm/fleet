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

  const renderCopyShowButtons = () => {
    return (
      <div className={`${baseClass}__action-overlay`}>
        {copyMessage && (
          <div
            className={`${baseClass}__copy-message`}
          >{`${copyMessage} `}</div>
        )}
        <div className={`${baseClass}__input-buttons`}>
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
        </div>
      </div>
    );
  };

  return (
    <div className={classNames}>
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__secret-input`}
        name={name}
        type={showSecret ? "text" : "password"}
        value={value}
      />
      {renderCopyShowButtons()}
    </div>
  );
};

export default InputFieldHiddenContent;
