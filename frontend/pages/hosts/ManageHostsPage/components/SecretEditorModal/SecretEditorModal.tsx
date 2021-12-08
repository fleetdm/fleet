import React, { useState, useCallback } from "react";
import { useSelector } from "react-redux";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IAddSecretModal {
  selectedTeam: number;
  onSaveSecret: (newEnrollSecret: string) => void;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  selectedSecret: IEnrollSecret | undefined;
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
}: IAddSecretModal): JSX.Element => {
  const [enrollSecretString, setEnrollSecretString] = useState<string>(
    selectedSecret ? selectedSecret.secret : randomSecretGenerator()
  );
  const [errors, setErrors] = useState<{ [key: string]: any }>({});

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
            label={"Secret"}
            type={"text"}
            value={enrollSecretString}
            onChange={onSecretChange}
            error={errors.secret}
            hint={"Must contain at least 32 characters."}
          />
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onSaveSecretClick} className="button button--brand">
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default SecretEditorModal;
