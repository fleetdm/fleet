import React, { useCallback, useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";
import { Link } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IApiError } from "interfaces/errors";
import { INewMembersBody, ITeam } from "interfaces/team";
import { IUpdateUserFormData, IUser, IUserFormErrors } from "interfaces/user";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import inviteAPI from "services/entities/invites";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import { DEFAULT_CREATE_USER_ERRORS } from "utilities/constants";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import CreateUserModal from "pages/admin/UserManagementPage/components/CreateUserModal";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import {
  IFormData,
  NewUserType,
} from "../../../UserManagementPage/components/UserForm/UserForm";
import userManagementHelpers from "../../../UserManagementPage/helpers";
import AddMemberModal from "./components/AddMemberModal";
import RemoveMemberModal from "./components/RemoveMemberModal";
import EmptyMembersTable from "./components/EmptyMembersTable";

import {
  generateColumnConfigs,
  generateDataSet,
  IMembersTableData,
} from "./MembersPageTableConfig";

const baseClass = "members";
const noMembersClass = "no-members";

const MembersPage = ({ location, router }: ITeamSubnavProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);
  const { config, currentUser, isGlobalAdmin, isPremiumTier } = useContext(
    AppContext
  );

  const { isRouteOk, isTeamAdmin, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  const smtpConfigured = config?.smtp_settings?.configured || false;
  const sesConfigured = config?.email?.backend === "ses" || false;
  const canUseSso = config?.sso_settings?.enable_sso || false;

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
    ["users", teamIdForApi, searchString],
    () =>
      usersAPI.loadAll({ teamId: teamIdForApi, globalFilter: searchString }),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data: IUser[]) => generateDataSet(teamIdForApi || 0, data), // Note: `enabled` condition ensures that teamIdForApi will be defined here but TypeScript can't infer type assertion
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
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: isRouteOk,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const currentTeamDetails = useMemo(
    () => teams?.find((team) => team.id === teamIdForApi),
    [teams, teamIdForApi]
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
      .removeMembers(teamIdForApi, removedUsers)
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
    userEditing?.id,
    userEditing?.name,
    teamIdForApi,
    renderFlash,
    currentUser,
    toggleRemoveMemberModal,
    refetchUsers,
  ]);

  const onAddMemberSubmit = useCallback(
    (newMembers: INewMembersBody) => {
      teamsAPI
        .addMembers(currentTeamDetails?.id, newMembers)
        .then(() => {
          const count = newMembers.users.length;
          renderFlash(
            "success",
            `${count} ${
              count === 1 ? "member" : "members"
            } successfully added to ${currentTeamDetails?.name}.`
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
    [
      currentTeamDetails?.id,
      currentTeamDetails?.name,
      renderFlash,
      toggleAddUserModal,
      refetchUsers,
    ]
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
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setCreateUserErrors({
              password: "Password is over the character limit.",
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
      const updatedAttrs: IUpdateUserFormData = userManagementHelpers.generateUpdateData(
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
                (thisTeam) => thisTeam.id === teamIdForApi
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
    [
      userEditing,
      renderFlash,
      currentUser,
      toggleEditMemberModal,
      teamIdForApi,
      refetchUsers,
    ]
  );

  const onActionSelection = useCallback(
    (action: string, user: IUser): void => {
      switch (action) {
        case "edit":
          toggleEditMemberModal(user);
          break;
        case "remove":
          toggleRemoveMemberModal(user);
          break;
        default:
      }
    },
    [toggleEditMemberModal, toggleRemoveMemberModal]
  );

  const columnConfigs = useMemo(
    () => generateColumnConfigs(onActionSelection),
    [onActionSelection]
  );

  if (!isRouteOk) {
    return <Spinner />;
  }

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
      (!currentTeamDetails && !isLoadingTeams && !isLoadingMembers) ? (
        <TableDataError />
      ) : (
        <TableContainer
          resultsTitle={"members"}
          columnConfigs={columnConfigs}
          data={members || []}
          isLoading={isLoadingMembers}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          actionButton={{
            name: isGlobalAdmin ? "add member" : "create user",
            buttonText: isGlobalAdmin ? "Add member" : "Create user",
            variant: "brand",
            onActionButtonClick: isGlobalAdmin
              ? toggleAddUserModal
              : toggleCreateMemberModal,
            hideButton: memberIds.length === 0 && searchString === "",
          }}
          onQueryChange={({ searchQuery }) => setSearchString(searchQuery)}
          inputPlaceHolder={"Search"}
          emptyComponent={() => (
            <EmptyMembersTable
              className={noMembersClass}
              isGlobalAdmin={!!isGlobalAdmin}
              isTeamAdmin={!!isTeamAdmin}
              searchString={searchString}
              toggleAddUserModal={toggleAddUserModal}
              toggleCreateMemberModal={toggleCreateMemberModal}
            />
          )}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={memberIds.length > 0 || searchString !== ""}
        />
      )}
      {showAddMemberModal && currentTeamDetails ? (
        <AddMemberModal
          team={currentTeamDetails}
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
          sesConfigured={sesConfigured}
          canUseSso={canUseSso}
          isSsoEnabled={userEditing?.sso_enabled}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          currentTeam={currentTeamDetails}
          isUpdatingUsers={isUpdatingMembers}
          isApiOnly={userEditing?.api_only || false}
        />
      )}
      {showCreateUserModal && currentTeamDetails && (
        <CreateUserModal
          createUserErrors={createUserErrors}
          onCancel={toggleCreateMemberModal}
          onSubmit={onCreateMemberSubmit}
          defaultGlobalRole={null}
          defaultTeamRole={"observer"}
          defaultTeams={[
            { id: currentTeamDetails.id, name: "", role: "observer" },
          ]}
          availableTeams={teams}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={smtpConfigured}
          sesConfigured={sesConfigured}
          canUseSso={canUseSso}
          currentTeam={currentTeamDetails}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          isUpdatingUsers={isUpdatingMembers}
        />
      )}
      {showRemoveMemberModal && currentTeamDetails && (
        <RemoveMemberModal
          memberName={userEditing?.name || ""}
          teamName={currentTeamDetails.name}
          isUpdatingMembers={isUpdatingMembers}
          onCancel={toggleRemoveMemberModal}
          onSubmit={onRemoveMemberSubmit}
        />
      )}
    </div>
  );
};

export default MembersPage;
