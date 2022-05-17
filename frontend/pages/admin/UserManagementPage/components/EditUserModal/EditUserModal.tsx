import React from "react";

import { ITeam } from "interfaces/team";
import { IUserFormErrors } from "interfaces/user";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import UserForm from "../UserForm";
import { IFormData } from "../UserForm/UserForm";

interface IEditUserModalProps {
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  defaultName?: string;
  defaultEmail?: string;
  defaultGlobalRole?: string | null;
  defaultTeamRole?: string;
  defaultTeams?: ITeam[];
  availableTeams: ITeam[];
  currentTeam?: ITeam;
  isPremiumTier: boolean;
  smtpConfigured: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  isSsoEnabled?: boolean; // corresponds to whether SSO is enabled for the individual user
  editUserErrors?: IUserFormErrors;
  isModifiedByGlobalAdmin?: boolean | false;
  isInvitePending?: boolean;
  isLoading: boolean;
}

const baseClass = "edit-user-modal";

const EditUserModal = ({
  onCancel,
  onSubmit,
  defaultName,
  defaultEmail,
  defaultGlobalRole,
  defaultTeamRole,
  defaultTeams,
  availableTeams,
  isPremiumTier,
  smtpConfigured,
  canUseSso,
  isSsoEnabled,
  currentTeam,
  editUserErrors,
  isModifiedByGlobalAdmin,
  isInvitePending,
  isLoading,
}: IEditUserModalProps): JSX.Element => {
  return (
    <Modal
      title="Edit user"
      onExit={onCancel}
      className={`${baseClass}__edit-user-modal`}
    >
      {isLoading ? (
        <Spinner />
      ) : (
        <UserForm
          createOrEditUserErrors={editUserErrors}
          defaultName={defaultName}
          defaultEmail={defaultEmail}
          defaultGlobalRole={defaultGlobalRole}
          defaultTeamRole={defaultTeamRole}
          defaultTeams={defaultTeams}
          onCancel={onCancel}
          onSubmit={onSubmit}
          availableTeams={availableTeams}
          submitText={"Save"}
          isPremiumTier={isPremiumTier}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          isSsoEnabled={isSsoEnabled}
          isModifiedByGlobalAdmin={isModifiedByGlobalAdmin}
          isInvitePending={isInvitePending}
          currentTeam={currentTeam}
        />
      )}
    </Modal>
  );
};

export default EditUserModal;
