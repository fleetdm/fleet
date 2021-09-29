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
  availableTeams: ITeam[];
  isPremiumTier: boolean;
  smtpConfigured: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isSsoEnabled?: boolean; // corresponds to whether SSO is enabled for the individual user
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
    availableTeams,
    isPremiumTier,
    smtpConfigured,
    canUseSso,
    isSsoEnabled,
  } = props;

  return (
    <Modal
      title="Edit user"
      onExit={onCancel}
      className={`${baseClass}__edit-user-modal`}
    >
      <UserForm
        defaultName={defaultName}
        defaultEmail={defaultEmail}
        defaultGlobalRole={defaultGlobalRole}
        defaultTeams={defaultTeams}
        onCancel={onCancel}
        onSubmit={onSubmit}
        availableTeams={availableTeams}
        submitText={"Save"}
        isPremiumTier={isPremiumTier}
        smtpConfigured={smtpConfigured}
        canUseSso={canUseSso}
        isSsoEnabled={isSsoEnabled}
      />
    </Modal>
  );
};

export default EditUserModal;
