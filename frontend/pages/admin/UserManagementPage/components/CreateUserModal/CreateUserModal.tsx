import React from "react";

import { ITeam } from "interfaces/team";
import { IUserFormErrors, UserRole } from "interfaces/user";
import Modal from "components/Modal";
import UserForm from "../UserForm";
import { IFormData } from "../UserForm/UserForm";

interface ICreateUserModalProps {
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
  createUserErrors?: IUserFormErrors;
}

const baseClass = "create-user-modal";

const CreateUserModal = ({
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
  createUserErrors,
}: ICreateUserModalProps): JSX.Element => {
  return (
    <Modal title="Create user" onExit={onCancel} className={baseClass}>
      <UserForm
        serverErrors={serverErrors}
        createOrEditUserErrors={createUserErrors}
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

export default CreateUserModal;
