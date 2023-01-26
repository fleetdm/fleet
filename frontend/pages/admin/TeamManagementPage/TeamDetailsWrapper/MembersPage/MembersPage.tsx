import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import PATHS from "router/paths";
import { IApiError } from "interfaces/errors";
import { IUser, IUserFormErrors } from "interfaces/user";
import { INewMembersBody, ITeam } from "interfaces/team";
import { IEmptyTableProps } from "interfaces/empty_table";
import { Link } from "react-router";
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import inviteAPI from "services/entities/invites";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import CreateUserModal from "pages/admin/UserManagementPage/components/CreateUserModal";
import { DEFAULT_CREATE_USER_ERRORS } from "utilities/constants";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import {
  IFormData,
  NewUserType,
} from "../../../UserManagementPage/components/UserForm/UserForm";
import userManagementHelpers from "../../../UserManagementPage/helpers";
import AddMemberModal from "./components/AddMemberModal";
import RemoveMemberModal from "./components/RemoveMemberModal";

import {
  generateTableHeaders,
  generateDataSet,
  IMembersTableData,
} from "./MembersPageTableConfig";

const baseClass = "members";
const noMembersClass = "no-members";

interface IMembersPageProps {
  params: {
    team_id: string;
  };
}

const MembersPage = ({
  params: { team_id },
}: IMembersPageProps): JSX.Element => {
  const teamId = parseInt(team_id, 10);

  const { renderFlash } = useContext(NotificationContext);
  const {
    config,
    currentUser,
    isGlobalAdmin,
    isPremiumTier,
    isTeamAdmin,
  } = useContext(AppContext);

  const smtpConfigured = config?.smtp_settings.configured || false;
  const canUseSso = config?.sso_settings.enable_sso || false;

  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [showRemoveMemberModal, setShowRemoveMemberModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [showCreateUserModal, setShowCreateUserModal] = useState(false);
  const [isUpdatingMembers, setIsUpdatingMembers] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();
  const [searchString, setSearchString] = useState("");
  const [createUserErrors, setCreateUserErrors] = useState<IUserFormErrors>(
    DEFAULT_CREATE_USER_ERRORS
  );
  const [editUserErrors, setEditUserErrors] = useState<IUserFormErrors>(
    DEFAULT_CREATE_USER_ERRORS
  );
  const [memberIds, setMemberIds] = useState<number[]>([]);
  const [currentTeam, setCurrentTeam] = useState<ITeam>();

  const toggleAddUserModal = useCallback(() => {
    setShowAddMemberModal(!showAddMemberModal);
  }, [showAddMemberModal, setShowAddMemberModal]);

  const toggleRemoveMemberModal = useCallback(
    (user?: IUser) => {
      setShowRemoveMemberModal(!showRemoveMemberModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
    },
    [showRemoveMemberModal, setShowRemoveMemberModal, setUserEditing]
  );

  // API CALLS

  const {
    data: members,
    isLoading: isLoadingMembers,
    error: loadingMembersError,
    refetch: refetchUsers,
  } = useQuery<IUser[], Error, IMembersTableData[]>(
    ["users", teamId, searchString],
    () => usersAPI.loadAll({ teamId, globalFilter: searchString }),
    {
      select: (data: IUser[]) => generateDataSet(teamId, data),
      onSuccess: (data) => {
        setMemberIds(data.map((member) => member.id));
      },
    }
  );

  const {
    data: teams,
    isLoading: isLoadingTeams,
    error: loadingTeamsError,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams", teamId],
    () => teamsAPI.loadAll(),
    {
      select: (data: ILoadTeamsResponse) => data.teams,
      onSuccess: (data) => {
        setCurrentTeam(data.find((team) => team.id === teamId));
      },
    }
  );

  // TOGGLE MODALS

  const toggleEditMemberModal = useCallback(
    (user?: IUser) => {
      setShowEditUserModal(!showEditUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
      setEditUserErrors(DEFAULT_CREATE_USER_ERRORS);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );

  const toggleCreateMemberModal = useCallback(() => {
    setShowCreateUserModal(!showCreateUserModal);
    setShowAddMemberModal(false);
  }, [showCreateUserModal, setShowCreateUserModal, setShowAddMemberModal]);

  // FUNCTIONS

  const onRemoveMemberSubmit = useCallback(() => {
    const removedUsers = { users: [{ id: userEditing?.id }] };
    setIsUpdatingMembers(true);
    teamsAPI
      .removeMembers(teamId, removedUsers)
      .then(() => {
        renderFlash(
          "success",
          `Successfully removed ${userEditing?.name || "member"}`
        );
        // If user removes self from team, redirect to home
        if (currentUser && currentUser.id === removedUsers.users[0].id) {
          window.location.href = "/";
        }
      })
      .catch(() =>
        renderFlash("error", "Unable to remove members. Please try again.")
      )
      .finally(() => {
        setIsUpdatingMembers(false);
        toggleRemoveMemberModal();
        refetchUsers();
      });
  }, [
    teamId,
    userEditing?.id,
    userEditing?.name,
    toggleRemoveMemberModal,
    refetchUsers,
  ]);

  const onAddMemberSubmit = useCallback(
    (newMembers: INewMembersBody) => {
      teamsAPI
        .addMembers(teamId, newMembers)
        .then(() => {
          const count = newMembers.users.length;
          renderFlash(
            "success",
            `${count} ${
              count === 1 ? "member" : "members"
            } successfully added to ${currentTeam?.name}.`
          );
        })
        .catch(() =>
          renderFlash("error", "Could not add members. Please try again.")
        )
        .finally(() => {
          toggleAddUserModal();
          refetchUsers();
        });
    },
    [teamId, toggleAddUserModal, currentTeam?.name, refetchUsers]
  );

  const onCreateMemberSubmit = (formData: IFormData) => {
    setIsUpdatingMembers(true);

    if (formData.newUserType === NewUserType.AdminInvited) {
      const requestData = {
        ...formData,
        invited_by: formData.currentUserId,
      };
      delete requestData.currentUserId;
      delete requestData.newUserType;
      delete requestData.password;
      inviteAPI
        .create(requestData)
        .then(() => {
          renderFlash(
            "success",
            `An invitation email was sent from ${config?.smtp_settings.sender_address} to ${formData.email}.`
          );
          refetchUsers();
          toggleCreateMemberModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (
            userErrors.data.errors?.[0].reason.includes(
              "a user with this account already exists"
            )
          ) {
            setCreateUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("Invite") &&
            userErrors.data.errors?.[0].reason.includes("already exists")
          ) {
            setCreateUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else {
            renderFlash("error", "Could not invite user. Please try again.");
          }
        })
        .finally(() => {
          setIsUpdatingMembers(false);
        });
    } else {
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId;
      delete requestData.newUserType;
      usersAPI
        .createUserWithoutInvitation(requestData)
        .then(() => {
          renderFlash("success", `Successfully created ${requestData.name}.`);
          refetchUsers();
          toggleCreateMemberModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors?.[0].reason.includes("Duplicate")) {
            setCreateUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("already invited")
          ) {
            setCreateUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
          }
        })
        .finally(() => {
          setIsUpdatingMembers(false);
        });
    }
  };

  const onEditMemberSubmit = useCallback(
    (formData: IFormData) => {
      const updatedAttrs = userManagementHelpers.generateUpdateData(
        userEditing as IUser,
        formData
      );

      setIsUpdatingMembers(true);

      const userName = userEditing?.name;

      userEditing &&
        usersAPI
          .update(userEditing.id, updatedAttrs)
          .then(() => {
            renderFlash(
              "success",
              `Successfully edited ${userName || "member"}.`
            );

            if (
              currentUser &&
              userEditing &&
              currentUser.id === userEditing.id
            ) {
              // If user edits self and removes "admin" role,
              // redirect to home
              const selectedTeam = formData.teams.filter(
                (thisTeam) => thisTeam.id === teamId
              );
              if (selectedTeam && selectedTeam[0].role !== "admin") {
                window.location.href = "/";
              }
            } else {
              refetchUsers();
            }
            toggleEditMemberModal();
          })
          .catch((userErrors: { data: IApiError }) => {
            if (userErrors.data.errors[0].reason.includes("already exists")) {
              setEditUserErrors({
                email: "A user with this email address already exists",
              });
            } else {
              renderFlash(
                "error",
                `Could not edit ${userName || "member"}. Please try again.`
              );
            }
          })
          .finally(() => {
            setIsUpdatingMembers(false);
          });
    },
    [toggleEditMemberModal, userEditing, refetchUsers]
  );

  const onActionSelection = (action: string, user: IUser): void => {
    switch (action) {
      case "edit":
        toggleEditMemberModal(user);
        break;
      case "remove":
        toggleRemoveMemberModal(user);
        break;
      default:
    }
  };

  const emptyState = () => {
    const emptyMembers: IEmptyTableProps = {
      iconName: "empty-members",
      header: "This team doesn't have any members yet.",
      info:
        "Expecting to see new team members listed here? Try again in a few seconds as the system catches up.",
    };
    if (searchString !== "") {
      delete emptyMembers.iconName;
      emptyMembers.header = "We couldn’t find any members.";
      emptyMembers.info =
        "Expecting to see members? Try again in a few seconds as the system catches up.";
    } else if (isGlobalAdmin) {
      emptyMembers.primaryButton = (
        <Button
          variant="brand"
          className={`${noMembersClass}__create-button`}
          onClick={toggleAddUserModal}
        >
          Add member
        </Button>
      );
    } else if (isTeamAdmin) {
      emptyMembers.primaryButton = (
        <Button
          variant="brand"
          className={`${noMembersClass}__create-button`}
          onClick={toggleCreateMemberModal}
        >
          Create user
        </Button>
      );
    }
    return emptyMembers;
  };

  const tableHeaders = generateTableHeaders(onActionSelection);

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__page-description`}>
        Users can either be a member of team(s) or a global user.{" "}
        {isGlobalAdmin && (
          <Link to={PATHS.ADMIN_USERS}>
            Manage users with global access here
          </Link>
        )}
      </p>
      {loadingMembersError ||
      loadingTeamsError ||
      (!currentTeam && !isLoadingTeams && !isLoadingMembers) ? (
        <TableDataError />
      ) : (
        <TableContainer
          resultsTitle={"members"}
          columns={tableHeaders}
          data={members || []}
          isLoading={isLoadingMembers}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          onActionButtonClick={
            isGlobalAdmin ? toggleAddUserModal : toggleCreateMemberModal
          }
          actionButtonText={isGlobalAdmin ? "Add member" : "Create user"}
          actionButtonVariant={"brand"}
          hideActionButton={memberIds.length === 0 && searchString === ""}
          onQueryChange={({ searchQuery }) => setSearchString(searchQuery)}
          inputPlaceHolder={"Search"}
          emptyComponent={() =>
            EmptyTable({
              iconName: emptyState().iconName,
              header: emptyState().header,
              info: emptyState().info,
              primaryButton: emptyState().primaryButton,
            })
          }
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={memberIds.length > 0 || searchString !== ""}
        />
      )}
      {showAddMemberModal && currentTeam ? (
        <AddMemberModal
          team={currentTeam}
          disabledMembers={memberIds}
          onCancel={toggleAddUserModal}
          onSubmit={onAddMemberSubmit}
          onCreateNewMember={toggleCreateMemberModal}
        />
      ) : null}
      {showEditUserModal && (
        <EditUserModal
          editUserErrors={editUserErrors}
          onCancel={toggleEditMemberModal}
          onSubmit={onEditMemberSubmit}
          defaultName={userEditing?.name}
          defaultEmail={userEditing?.email}
          defaultGlobalRole={userEditing?.global_role || null}
          defaultTeamRole={userEditing?.role}
          defaultTeams={userEditing?.teams}
          availableTeams={teams || []}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          isSsoEnabled={userEditing?.sso_enabled}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          currentTeam={currentTeam}
          isUpdatingUsers={isUpdatingMembers}
        />
      )}
      {showCreateUserModal && (
        <CreateUserModal
          createUserErrors={createUserErrors}
          onCancel={toggleCreateMemberModal}
          onSubmit={onCreateMemberSubmit}
          defaultGlobalRole={null}
          defaultTeamRole={"observer"}
          defaultTeams={[{ id: teamId, name: "", role: "observer" }]}
          availableTeams={teams}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          currentTeam={currentTeam}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          isUpdatingUsers={isUpdatingMembers}
        />
      )}
      {showRemoveMemberModal && currentTeam && (
        <RemoveMemberModal
          memberName={userEditing?.name || ""}
          teamName={currentTeam.name}
          isUpdatingMembers={isUpdatingMembers}
          onCancel={toggleRemoveMemberModal}
          onSubmit={onRemoveMemberSubmit}
        />
      )}
    </div>
  );
};

export default MembersPage;
