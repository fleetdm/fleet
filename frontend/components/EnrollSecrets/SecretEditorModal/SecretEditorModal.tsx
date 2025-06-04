import React, { useState } from "react";

import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

interface ISecretEditorModalProps {
  selectedTeam: number;
  onSaveSecret: (newEnrollSecret: string) => void;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  selectedSecret: IEnrollSecret | undefined;
  isUpdatingSecret: boolean;
}

const baseClass = "secret-editor-modal";

const randomSecretGenerator = () => {
  const randomChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
  let result = "";
  for (let i = 0; i < 32; i += 1) {
    result += randomChars.charAt(
      Math.floor(Math.random() * randomChars.length)
    );
  }
  return result;
};

const SecretEditorModal = ({
  onSaveSecret,
  selectedTeam,
  teams,
  toggleSecretEditorModal,
  selectedSecret,
  isUpdatingSecret,
}: ISecretEditorModalProps): JSX.Element => {
  const [enrollSecretString, setEnrollSecretString] = useState(
    selectedSecret ? selectedSecret.secret : randomSecretGenerator()
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team" };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  const onSecretChange = (value: string) => {
    if (value.length < 32) {
      setErrors({
        secret: "Secret",
      });
    } else {
      setErrors({});
    }
    setEnrollSecretString(value);
  };

  const onSaveSecretClick = () => {
    if (enrollSecretString.length < 32) {
      setErrors({
        secret: "Secret",
      });
    } else {
      setErrors({});
      onSaveSecret(enrollSecretString);
    }
  };

  return (
    <Modal
      onExit={toggleSecretEditorModal}
      onEnter={onSaveSecretClick}
      title={selectedSecret ? "Edit secret" : "Add secret"}
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__description`}>
          Create or edit the generated secret to enroll hosts to{" "}
          <b>{renderTeam()?.name}</b>:
        </div>
        <div className={`${baseClass}__secret-wrapper`}>
          <InputField
            inputWrapperClass={`${baseClass}__secret-input`}
            name="osqueryd-secret"
            label="Secret"
            type="text"
            value={enrollSecretString}
            onChange={onSecretChange}
            error={errors.secret}
            helpText="Must contain at least 32 characters."
          />
        </div>
        <div className="modal-cta-wrap">
          <Button
            onClick={onSaveSecretClick}
            className="save-loading"
            isLoading={isUpdatingSecret}
          >
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default SecretEditorModal;
