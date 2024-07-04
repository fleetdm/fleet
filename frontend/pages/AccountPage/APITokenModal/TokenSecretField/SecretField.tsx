import React, { useState } from "react";

import { stringToClipboard } from "utilities/copy_text";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "secret-field";

interface ISecretFieldProps {
  secret: string | null;
}
const SecretField = ({ secret }: ISecretFieldProps): JSX.Element | null => {
  const [showSecret, setShowSecret] = useState(false);
  const [copyMessage, setCopyMessage] = useState("");

  const onCopySecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(secret)
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
      <div className="buttons">
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
      </div>
    );
  };

  return (
    <div className={`${baseClass}__secret`} key={secret}>
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__secret-input`}
        name="secret-field"
        type={showSecret ? "text" : "password"}
        value={secret}
      />
      {renderCopyShowButtons()}
    </div>
  );
};

export default SecretField;
