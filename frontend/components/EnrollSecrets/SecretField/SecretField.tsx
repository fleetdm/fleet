import React, { useState } from "react";

import { stringToClipboard } from "utilities/copy_text";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import EyeIcon from "../../../../assets/images/icon-eye-16x16@2x.png";
import ClipboardIcon from "../../../../assets/images/icon-copy-clipboard-fleet-blue-20x20@2x.png";

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
            <img src={ClipboardIcon} alt="copy to clipboard" />
          </Button>
          <a
            href="#showSecret"
            onClick={onToggleSecret}
            className={`${baseClass}__show-secret`}
          >
            <img src={EyeIcon} alt="show/hide" />
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
