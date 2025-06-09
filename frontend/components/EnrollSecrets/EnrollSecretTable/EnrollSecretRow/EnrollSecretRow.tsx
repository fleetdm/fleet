import React from "react";
import { uniqueId } from "lodash";

import { IEnrollSecret } from "interfaces/enroll_secret";

import Button from "components/buttons/Button";
import InputFieldHiddenContent from "components/forms/fields/InputFieldHiddenContent";
import Icon from "components/Icon";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

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

  const renderEditDeleteButtons = () => (
    <GitOpsModeTooltipWrapper
      tipOffset={8}
      renderChildren={(disableChildren) => (
        <div className={`${baseClass}__edit-delete-btns`}>
          <Button
            disabled={disableChildren}
            onClick={onEditSecretClick}
            className={`${baseClass}__edit-secret-icon`}
            variant="icon"
          >
            <Icon name="pencil" />
          </Button>
          <Button
            onClick={onDeleteSecretClick}
            disabled={disableChildren}
            className={`${baseClass}__delete-secret-icon`}
            variant="icon"
          >
            <Icon name="trash" />
          </Button>
        </div>
      )}
    />
  );

  return (
    <div
      className={`${baseClass}__secret`}
      key={uniqueId()}
      data-testid="osquery-secret"
    >
      <InputFieldHiddenContent
        name={`osqueryd-secret-${uniqueId()}`}
        value={secret.secret}
      />
      {toggleSecretEditorModal &&
        toggleDeleteSecretModal &&
        renderEditDeleteButtons()}
    </div>
  );
};

export default EnrollSecretRow;
