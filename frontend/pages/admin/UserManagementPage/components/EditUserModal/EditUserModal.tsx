import React from "react";

import { ITeam } from "interfaces/team";
import Modal from "components/modals/Modal";
import UserForm from "../UserForm";
import { IFormData } from "../UserForm/UserForm";

interface IEditUserModalProps {
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  defaultName?: string;
  defaultEmail?: string;
  defaultGlobalRole?: string | null;
  defaultTeams?: ITeam[];
  defaultSSOEnabled?: boolean;
  availableTeams: ITeam[];
  validationErrors: any[];
  isBasicTier: boolean;
  smtpConfigured: boolean;
}

const baseClass = "edit-user-modal";

const EditUserModal = (props: IEditUserModalProps): JSX.Element => {
  const {
    onCancel,
    onSubmit,
    defaultName,
    defaultEmail,
    defaultGlobalRole,
    defaultTeams,
    defaultSSOEnabled,
    availableTeams,
    isBasicTier,
    validationErrors,
    smtpConfigured,
  } = props;

  return (
    <Modal
      title="Edit user"
      onExit={onCancel}
      className={`${baseClass}__edit-user-modal`}
    >
      <UserForm
        validationErrors={validationErrors}
        defaultName={defaultName}
        defaultEmail={defaultEmail}
        defaultGlobalRole={defaultGlobalRole}
        defaultTeams={defaultTeams}
        onCancel={onCancel}
        onSubmit={onSubmit}
        canUseSSO={defaultSSOEnabled}
        availableTeams={availableTeams}
        submitText={"Save"}
        isBasicTier={isBasicTier}
        smtpConfigured={smtpConfigured}
      />
    </Modal>
  );
};

export default EditUserModal;
