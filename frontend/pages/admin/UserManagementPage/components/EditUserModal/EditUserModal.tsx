import React from "react";

import { ITeam } from "interfaces/team";
import { IUser, IUserFormErrors } from "interfaces/user";
import { IInvite } from "interfaces/invite";
import Modal from "components/Modal";
import UserForm from "../UserForm";
import { IFormData } from "../UserForm/UserForm";

interface IEditUserModalProps {
  onCancel: () => void;
  onSubmit: (formData: IFormData) => void;
  userToEdit?: IUser | IInvite;
  currentUser?: IUser;
  availableTeams: ITeam[];
  currentTeam?: ITeam;
  isPremiumTier: boolean;
  smtpConfigured: boolean;
  sesConfigured: boolean;
  canUseSso: boolean; // corresponds to whether SSO is enabled for the organization
  editUserErrors?: IUserFormErrors;
  isModifiedByGlobalAdmin?: boolean | false;
  isInvitePending?: boolean;
  isUpdatingUsers: boolean;
}

const baseClass = "edit-user-modal";

const EditUserModal = ({
  onCancel,
  onSubmit,
  userToEdit,
  currentUser,
  availableTeams,
  isPremiumTier,
  smtpConfigured,
  sesConfigured,
  canUseSso,
  currentTeam,
  editUserErrors,
  isModifiedByGlobalAdmin,
  isInvitePending,
  isUpdatingUsers,
}: IEditUserModalProps): JSX.Element => {
  const isUser = (userOrInvite: IUser | IInvite): userOrInvite is IUser => {
    return (userToEdit as IInvite).invited_by === undefined;
  };

  return (
    <Modal
      title="Edit user"
      onExit={onCancel}
      className={`${baseClass}__edit-user-modal`}
    >
      <UserForm
        createOrEditUserErrors={editUserErrors}
        userToEditId={userToEdit?.id}
        currentUserId={currentUser?.id}
        defaultName={userToEdit?.name}
        defaultEmail={userToEdit?.email}
        defaultGlobalRole={userToEdit?.global_role || null}
        defaultTeamRole={
          (!!userToEdit && isUser(userToEdit) && userToEdit?.role) || undefined
        }
        defaultTeams={userToEdit?.teams}
        onCancel={onCancel}
        onSubmit={onSubmit}
        availableTeams={availableTeams}
        submitText="Save"
        isPremiumTier={isPremiumTier}
        smtpConfigured={smtpConfigured}
        sesConfigured={sesConfigured}
        canUseSso={canUseSso}
        isSsoEnabled={userToEdit?.sso_enabled}
        isApiOnly={userToEdit?.api_only || false}
        isModifiedByGlobalAdmin={isModifiedByGlobalAdmin}
        isInvitePending={isInvitePending}
        currentTeam={currentTeam}
        isUpdatingUsers={isUpdatingUsers}
      />
    </Modal>
  );
};

export default EditUserModal;
