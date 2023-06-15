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

  const renderLabel = () => {
    return (
      <span className={`${baseClass}`}>
        <span className="buttons">
          {copyMessage && (
            <span className="copy-message">{`${copyMessage} `}</span>
          )}
          <Button
            variant="unstyled"
            className={`${baseClass}__secret-copy-icon`}
            onClick={onCopySecret}
          >
            <Icon name="copy" />
          </Button>
          <a
            href="#showSecret"
            onClick={onToggleSecret}
            className={`${baseClass}__show-secret`}
          >
            <Icon name="eye" />
          </a>
        </span>
      </span>
    );
  };

  return (
    <div className={`${baseClass}__secret`} key={secret}>
      <InputField
        disabled
        inputWrapperClass={`${baseClass}__secret-input`}
        name="secret-field"
        label={renderLabel()}
        type={showSecret ? "text" : "password"}
        value={secret}
      />
    </div>
  );
};

export default SecretField;
