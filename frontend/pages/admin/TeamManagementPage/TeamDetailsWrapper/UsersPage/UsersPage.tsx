import React, { useCallback, useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";
import { Link } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IApiError } from "interfaces/errors";
import { INewTeamUsersBody, ITeam } from "interfaces/team";
import { IUpdateUserFormData, IUser, IUserFormErrors } from "interfaces/user";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import PATHS from "router/paths";
import usersAPI from "services/entities/users";
import inviteAPI from "services/entities/invites";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import { DEFAULT_USER_FORM_ERRORS } from "utilities/constants";

import TableContainer from "components/TableContainer";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import TableCount from "components/TableContainer/TableCount";
import AddUserModal from "pages/admin/UserManagementPage/components/AddUserModal";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import {
  IUserFormData,
  NewUserType,
} from "../../../UserManagementPage/components/UserForm/UserForm";
import userManagementHelpers from "../../../UserManagementPage/helpers";
import EmptyMembersTable from "./components/EmptyUsersTable";
import AddUsersModal from "./components/AddUsersModal/AddUsersModal";
import RemoveUserModal from "./components/RemoveUserModal/RemoveUserModal";

import {
  generateColumnConfigs,
  generateDataSet,
  ITeamUsersTableData,
} from "./UsersPageTableConfig";

const baseClass = "team-users";
const noUsersClass = "no-team-users";

const UsersPage = ({ location, router }: ITeamSubnavProps): JSX.Element => {
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

  const [showAddUserModal, setShowAddUserModal] = useState(false);
  const [showRemoveUserModal, setShowRemoveUserModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [showCreateUserModal, setShowCreateUserModal] = useState(false);
  const [isUpdatingUsers, setIsUpdatingUsers] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();
  const [searchString, setSearchString] = useState("");
  const [addUserErrors, setAddUserErrors] = useState<IUserFormErrors>(
    DEFAULT_USER_FORM_ERRORS
  );
  const [editUserErrors, setEditUserErrors] = useState<IUserFormErrors>(
    DEFAULT_USER_FORM_ERRORS
  );

  const toggleAddUserModal = useCallback(() => {
    setShowAddUserModal(!showAddUserModal);
  }, [showAddUserModal, setShowAddUserModal]);

  const toggleRemoveUserModal = useCallback(
    (user?: IUser) => {
      setShowRemoveUserModal(!showRemoveUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
    },
    [showRemoveUserModal, setShowRemoveUserModal, setUserEditing]
  );

  // API CALLS

  const {
    data: teamUsers,
    isLoading: isLoadingUsers,
    error: loadingUsersError,
    refetch: refetchUsers,
  } = useQuery<IUser[], Error, ITeamUsersTableData[]>(
    ["users", teamIdForApi, searchString],
    () =>
      usersAPI.loadAll({ teamId: teamIdForApi, globalFilter: searchString }),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data: IUser[]) => generateDataSet(teamIdForApi || 0, data), // Note: `enabled` condition ensures that teamIdForApi will be defined here but TypeScript can't infer type assertion
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

  const toggleEditUserModal = useCallback(
    (user?: IUser) => {
      setShowEditUserModal(!showEditUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
      setEditUserErrors(DEFAULT_USER_FORM_ERRORS);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );

  const toggleCreateUserModal = useCallback(() => {
    setShowCreateUserModal(!showCreateUserModal);
    setShowAddUserModal(false);
  }, [showCreateUserModal, setShowCreateUserModal, setShowAddUserModal]);

  // FUNCTIONS

  const onRemoveUserSubmit = useCallback(() => {
    const removedUsers = { users: [{ id: userEditing?.id }] };
    setIsUpdatingUsers(true);
    teamsAPI
      .removeUsers(teamIdForApi, removedUsers)
      .then(() => {
        renderFlash(
          "success",
          `Successfully removed ${userEditing?.name || "user"}`
        );
        // If user removes self from team, redirect to home
        if (currentUser && currentUser.id === removedUsers.users[0].id) {
          window.location.href = "/";
        }
      })
      .catch(() =>
        renderFlash("error", "Unable to remove users. Please try again.")
      )
      .finally(() => {
        setIsUpdatingUsers(false);
        toggleRemoveUserModal();
        refetchUsers();
      });
  }, [
    userEditing?.id,
    userEditing?.name,
    teamIdForApi,
    renderFlash,
    currentUser,
    toggleRemoveUserModal,
    refetchUsers,
  ]);

  // GLobal admins get Add USER
  const onAddUserSubmit = useCallback(
    (newUsers: INewTeamUsersBody) => {
      debugger;
      teamsAPI
        .addUsers(currentTeamDetails?.id, newUsers)
        .then(() => {
          const count = newUsers.users.length;
          renderFlash(
            "success",
            `${count} ${count === 1 ? "user" : "users"} successfully added to ${
              currentTeamDetails?.name
            }.`
          );
        })
        .catch(() =>
          renderFlash("error", "Could not add users. Please try again.")
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

  // NON-global admins get CREATE USER
  const onCreateUserSubmit = (formData: IUserFormData) => {
    setIsUpdatingUsers(true);
    debugger;

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
          const senderAddressMessage = config?.smtp_settings?.sender_address
            ? ` from ${config?.smtp_settings?.sender_address}`
            : "";
          renderFlash(
            "success",
            `An invitation email was sent${senderAddressMessage} to ${formData.email}.`
          );
          refetchUsers();
          toggleCreateUserModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (
            userErrors.data.errors?.[0].reason.includes(
              "a user with this account already exists"
            )
          ) {
            setAddUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("Invite") &&
            userErrors.data.errors?.[0].reason.includes("already exists")
          ) {
            setAddUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else {
            renderFlash("error", "Could not invite user. Please try again.");
          }
        })
        .finally(() => {
          setIsUpdatingUsers(false);
        });
    } else {
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId;
      delete requestData.newUserType;
      debugger;
      usersAPI
        .createUserWithoutInvitation(requestData)
        .then(() => {
          renderFlash("success", `Successfully created ${requestData.name}.`);
          refetchUsers();
          toggleCreateUserModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          debugger;
          if (userErrors.data.errors?.[0].reason.includes("Duplicate")) {
            setAddUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("already invited")
          ) {
            setAddUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setAddUserErrors({
              password: "Password is over the character limit.",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
          }
        })
        .finally(() => {
          toggleAddUserModal();
          setIsUpdatingUsers(false);
        });
    }
  };

  const onEditUserSubmit = useCallback(
    (formData: IUserFormData) => {
      debugger;
      const updatedAttrs: IUpdateUserFormData = userManagementHelpers.generateUpdateData(
        userEditing as IUser,
        formData
      );

      setIsUpdatingUsers(true);

      const userName = userEditing?.name;

      userEditing &&
        usersAPI
          .update(userEditing.id, updatedAttrs)
          .then(() => {
            renderFlash(
              "success",
              `Successfully edited ${userName || "user"}.`
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
            toggleEditUserModal();
          })
          .catch((userErrors: { data: IApiError }) => {
            if (userErrors.data.errors[0].reason.includes("already exists")) {
              setEditUserErrors({
                email: "A user with this email address already exists",
              });
            } else {
              renderFlash(
                "error",
                `Could not edit ${userName || "user"}. Please try again.`
              );
            }
          })
          .finally(() => {
            setIsUpdatingUsers(false);
          });
    },
    [
      userEditing,
      renderFlash,
      currentUser,
      toggleEditUserModal,
      teamIdForApi,
      refetchUsers,
    ]
  );

  const onActionSelection = useCallback(
    (action: string, user: IUser): void => {
      switch (action) {
        case "edit":
          toggleEditUserModal(user);
          break;
        case "remove":
          toggleRemoveUserModal(user);
          break;
        default:
      }
    },
    [toggleEditUserModal, toggleRemoveUserModal]
  );

  const renderUsersCount = useCallback(() => {
    if (teamUsers?.length === 0 && searchString === "") {
      return <></>;
    }

    return <TableCount name="users" count={teamUsers?.length} />;
  }, [teamUsers?.length]);

  const columnConfigs = useMemo(
    () => generateColumnConfigs(onActionSelection),
    [onActionSelection]
  );

  if (!isRouteOk) {
    return <Spinner />;
  }

  const userIds = teamUsers ? teamUsers.map((user) => user.id) : [];

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__page-description`}>
        Manage users with access to this team.{" "}
        {isGlobalAdmin && (
          <Link to={PATHS.ADMIN_USERS}>
            Manage users with global access here
          </Link>
        )}
      </p>
      {loadingUsersError ||
      loadingTeamsError ||
      (!currentTeamDetails && !isLoadingTeams && !isLoadingUsers) ? (
        <TableDataError />
      ) : (
        <TableContainer
          resultsTitle="users"
          columnConfigs={columnConfigs}
          data={teamUsers || []}
          isLoading={isLoadingUsers}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          actionButton={{
            name: isGlobalAdmin ? "add user" : "create user",
            buttonText: isGlobalAdmin ? "Add users" : "Create user",
            variant: "brand",
            onActionButtonClick: isGlobalAdmin
              ? toggleAddUserModal
              : toggleCreateUserModal,
            hideButton: userIds.length === 0 && searchString === "",
          }}
          onQueryChange={({ searchQuery }) => setSearchString(searchQuery)}
          inputPlaceHolder="Search"
          emptyComponent={() => (
            <EmptyMembersTable
              className={noUsersClass}
              isGlobalAdmin={!!isGlobalAdmin}
              isTeamAdmin={!!isTeamAdmin}
              searchString={searchString}
              toggleAddUserModal={toggleAddUserModal}
              toggleCreateMemberModal={toggleCreateUserModal}
            />
          )}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable={userIds.length > 0 || searchString !== ""}
          renderCount={renderUsersCount}
        />
      )}
      {showAddUserModal && currentTeamDetails ? (
        <AddUsersModal
          team={currentTeamDetails}
          disabledUsers={userIds}
          onCancel={toggleAddUserModal}
          onSubmit={onAddUserSubmit}
          onCreateNewTeamUser={toggleCreateUserModal}
        />
      ) : null}
      {showEditUserModal && (
        <EditUserModal
          editUserErrors={editUserErrors}
          onCancel={toggleEditUserModal}
          onSubmit={onEditUserSubmit}
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
          isMfaEnabled={userEditing?.mfa_enabled}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          currentTeam={currentTeamDetails}
          isUpdatingUsers={isUpdatingUsers}
          isApiOnly={userEditing?.api_only || false}
        />
      )}
      {showCreateUserModal && currentTeamDetails && (
        <AddUserModal
          addUserErrors={addUserErrors}
          onCancel={toggleCreateUserModal}
          onSubmit={onCreateUserSubmit}
          defaultGlobalRole={null}
          defaultTeamRole="Observer"
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
          isUpdatingUsers={isUpdatingUsers}
        />
      )}
      {showRemoveUserModal && currentTeamDetails && (
        <RemoveUserModal
          userName={userEditing?.name || ""}
          teamName={currentTeamDetails.name}
          isUpdatingUsers={isUpdatingUsers}
          onCancel={toggleRemoveUserModal}
          onSubmit={onRemoveUserSubmit}
        />
      )}
    </div>
  );
};

export default UsersPage;
