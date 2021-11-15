import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
// @ts-ignore
import { stringToClipboard } from "utilities/copy_text";
import { IEnrollSecret } from "interfaces/enroll_secret";
import EyeIcon from "../../../../assets/images/icon-eye-16x16@2x.png";
import EditIcon from "../../../../assets/images/icon-pencil-14x14@2x.png";
import DeleteIcon from "../../../../assets/images/icon-trash-14x14@2x.png";

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
  const [showSecret, setShowSecret] = useState<boolean>(false);
  const [copyMessage, setCopyMessage] = useState<string>("");

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

  const renderLabel = () => {
    return (
      <span className={`${baseClass}__name`}>
        <span className="buttons">
          {copyMessage && <span>{`${copyMessage} `}</span>}
          <Button
            variant="unstyled"
            className={`${baseClass}__secret-copy-icon`}
            onClick={onCopySecret}
          >
            <FleetIcon name="clipboard" />
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
    <div className={`${baseClass}__secret`} key={secret.secret}>
      <InputField
        disabled
        inputWrapperClass={`${baseClass}__secret-input`}
        name="osqueryd-secret"
        label={renderLabel()}
        type={showSecret ? "text" : "password"}
        value={secret.secret}
      />
      {toggleSecretEditorModal && toggleDeleteSecretModal ? (
        <>
          <Button
            onClick={onEditSecretClick}
            className={`${baseClass}__edit-secret-btn`}
            variant="text-icon"
          >
            <>
              <img src={EditIcon} alt="Edit secret icon" />
            </>
          </Button>
          <Button
            onClick={onDeleteSecretClick}
            className={`${baseClass}__delete-secret-btn`}
            variant="text-icon"
          >
            <>
              <img src={DeleteIcon} alt="Delete secret icon" />
            </>
          </Button>
        </>
      ) : null}
    </div>
  );
};

export default EnrollSecretRow;
