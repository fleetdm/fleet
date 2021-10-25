import React, { useState, useCallback } from "react";
import { useSelector } from "react-redux";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IAddSecretModal {
  selectedTeam: number;
  onReturnToApp: () => void;
  onSaveSecret: () => void;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  selectedSecret: IEnrollSecret | undefined;
  setNewEnrollSecretString: React.Dispatch<
    React.SetStateAction<string | undefined>
  >;
}

interface IRootState {
  app: {
    enrollSecret: IEnrollSecret[];
  };
}

const baseClass = "secret-editor-modal";

const randomSecretGenerator = () => {
  const randomChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
  let result = "";
  for (var i = 0; i < 32; i++) {
    result += randomChars.charAt(
      Math.floor(Math.random() * randomChars.length)
    );
  }
  return result;
};

const SecretEditorModal = ({
  onReturnToApp, // do we want to return to app or back to previous modal?
  onSaveSecret,
  selectedTeam,
  teams,
  toggleSecretEditorModal,
  selectedSecret,
  setNewEnrollSecretString,
}: IAddSecretModal): JSX.Element => {
  const globalSecret = useSelector(
    (state: IRootState) => state.app.enrollSecret
  );

  const [enrollSecretString, setEnrollSecretString] = useState<string>(
    selectedSecret ? selectedSecret.secret : randomSecretGenerator()
  );
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team", secrets: globalSecret };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

  const onSecretChange = (value: string) => {
    setEnrollSecretString(value);
  };

  const onSaveSecretClick = () => {
    setNewEnrollSecretString(enrollSecretString);
    onSaveSecret;
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
            // value={selectedSecret ? selectedSecret.secret : enrollSecretString}
            value={enrollSecretString}
            onChange={onSecretChange}
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
