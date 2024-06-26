import React, { useState } from "react";
import { uniqueId } from "lodash";

import { stringToClipboard } from "utilities/copy_text";
import { IEnrollSecret } from "interfaces/enroll_secret";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Icon from "components/Icon";

const baseClass = "enroll-secrets";

interface IEnrollSecretRowProps {
  secret: IEnrollSecret;
  toggleSecretEditorModal?: () => void;
  toggleDeleteSecretModal?: () => void;
  setSelectedSecret?: React.Dispatch<
    React.SetStateAction<IEnrollSecret | undefined>
  >;
}
const EnrollSecretRow = ({
  secret,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
}: IEnrollSecretRowProps): JSX.Element | null => {
  const [showSecret, setShowSecret] = useState(false);
  const [copyMessage, setCopyMessage] = useState("");

  const onCopySecret = (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(secret.secret)
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

  const onEditSecretClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    if (toggleSecretEditorModal && setSelectedSecret) {
      setSelectedSecret(secret);
      toggleSecretEditorModal();
    }
  };

  const onDeleteSecretClick = (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    if (toggleDeleteSecretModal && setSelectedSecret) {
      setSelectedSecret(secret);
      toggleDeleteSecretModal();
    }
  };

  const renderCopyShowButtons = () => {
    return (
      <div className={`${baseClass}__action-overlay`}>
        {copyMessage && (
          <div
            className={`${baseClass}__copy-message`}
          >{`${copyMessage} `}</div>
        )}
        <div className="buttons">
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

  const renderEditDeleteButtons = () => (
    <div className="buttons">
      <Button
        onClick={onEditSecretClick}
        className={`${baseClass}__edit-secret-icon`}
        variant="text-icon"
      >
        <Icon name="pencil" />
      </Button>
      <Button
        onClick={onDeleteSecretClick}
        className={`${baseClass}__delete-secret-icon`}
        variant="text-icon"
      >
        <Icon name="trash" />
      </Button>
    </div>
  );

  return (
    <div
      className={`${baseClass}__secret`}
      key={uniqueId()}
      data-testid="osquery-secret"
    >
      {/* TODO: replace with InputFieldHiddenContent component */}
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__secret-input`}
        name={`osqueryd-secret-${uniqueId()}`}
        type={showSecret ? "text" : "password"}
        value={secret.secret}
      />
      {renderCopyShowButtons()}
      {toggleSecretEditorModal &&
        toggleDeleteSecretModal &&
        renderEditDeleteButtons()}
    </div>
  );
};

export default EnrollSecretRow;
