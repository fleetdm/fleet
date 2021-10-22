import React from "react";
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
  isPremiumTier: boolean;
  teams: ITeam[];
  toggleSecretEditorModal: () => void;
  selectedSecret: IEnrollSecret | undefined;
}

interface IRootState {
  app: {
    enrollSecret: IEnrollSecret[];
  };
}

const baseClass = "secret-editor-modal";

const AddSecretModal = ({
  onReturnToApp,
  onSaveSecret,
  selectedTeam,
  isPremiumTier,
  teams,
  toggleSecretEditorModal,
  selectedSecret,
}: IAddSecretModal): JSX.Element => {
  const globalSecret = useSelector(
    (state: IRootState) => state.app.enrollSecret
  );

  const [enrollSecretString, setEnrollSecretString] = useState<string>("");
  const renderTeam = () => {
    if (typeof selectedTeam === "string") {
      selectedTeam = parseInt(selectedTeam, 10);
    }

    if (selectedTeam === 0) {
      return { name: "No team", secrets: globalSecret };
    }
    return teams.find((team) => team.id === selectedTeam);
  };

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

  const randomGeneratedSecret = randomSecretGenerator();

  if (!selectedSecret) {
    setEnrollSecretString(randomGeneratedSecret);
  }

  const onSecretChange = (value: string) => {
    setEnrollSecretString(value);
  };

  return (
    <Modal
      onExit={onReturnToApp}
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
            value={
              selectedSecret ? selectedSecret.secret : randomGeneratedSecret
            }
            onChange={onSecretChange}
          />
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button onClick={onSaveSecret} className="button button--brand">
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default AddSecretModal;
