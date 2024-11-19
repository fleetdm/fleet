import React from "react";

import { ITeam } from "interfaces/team";
import { IUserFormErrors, UserRole } from "interfaces/user";
import Modal from "components/Modal";
import UserForm from "../UserForm";
import { IFormData } from "../UserForm/UserForm";

interface IAddUserModalProps {
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  defaultGlobalRole?: UserRole | null;
  defaultTeamRole?: UserRole;
  defaultTeams?: ITeam[];
  availableTeams?: ITeam[];
  isPremiumTier: boolean;
  smtpConfigured: boolean;
  sesConfigured: boolean;
  currentTeam?: ITeam;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isModifiedByGlobalAdmin?: boolean | false;
  isUpdatingUsers?: boolean | false;
  serverErrors?: { base: string; email: string };
  addUserErrors?: IUserFormErrors;
}

const baseClass = "add-user-modal";

const AddUserModal = ({
  onCancel,
  onSubmit,
  currentTeam,
  defaultGlobalRole,
  defaultTeamRole,
  defaultTeams,
  availableTeams,
  isPremiumTier,
  smtpConfigured,
  sesConfigured,
  canUseSso,
  isModifiedByGlobalAdmin,
  isUpdatingUsers,
  serverErrors,
  addUserErrors,
}: IAddUserModalProps): JSX.Element => {
  return (
    <Modal
      title="Add user"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <UserForm
        serverErrors={serverErrors}
        createOrEditUserErrors={addUserErrors}
        defaultGlobalRole={defaultGlobalRole}
        defaultTeamRole={defaultTeamRole}
        defaultTeams={defaultTeams}
        onCancel={onCancel}
        onSubmit={onSubmit}
        availableTeams={availableTeams || []}
        submitText="Create"
        isPremiumTier={isPremiumTier}
        smtpConfigured={smtpConfigured}
        sesConfigured={sesConfigured}
        canUseSso={canUseSso}
        isModifiedByGlobalAdmin={isModifiedByGlobalAdmin}
        currentTeam={currentTeam}
        isNewUser
        isUpdatingUsers={isUpdatingUsers}
      />
    </Modal>
  );
};

export default AddUserModal;
