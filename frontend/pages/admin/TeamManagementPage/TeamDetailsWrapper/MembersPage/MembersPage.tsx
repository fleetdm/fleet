import React, { useCallback, useContext, useState } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";

// @ts-ignore
import PATHS from "router/paths";
import { IApiError } from "interfaces/errors";
import { IUser } from "interfaces/user";
import { INewMembersBody, ITeam } from "interfaces/team";
import { Link } from "react-router";
import { AppContext } from "context/app";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import usersAPI from "services/entities/users";
import inviteAPI from "services/entities/invites";
import teamsAPI from "services/entities/teams";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
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

interface IFetchParams {
  pageIndex?: number;
  pageSize?: number;
  searchQuery?: string;
}

interface ITeamsResponse {
  teams: ITeam[];
}

// This is used to cache the table query data and make a request for the
// members data at a future time. Practically, this allows us to re-fetch the users
// with the same table query params after we have made an edit to a user.
let tableQueryData = {};

const MembersPage = ({
  params: { team_id },
}: IMembersPageProps): JSX.Element => {
  const teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();

  const { config, isGlobalAdmin, currentUser, isPremiumTier } = useContext(
    AppContext
  );

  const smtpConfigured = config?.configured || false;
  const canUseSso = config?.enable_sso || false;

  const [showAddMemberModal, setShowAddMemberModal] = useState<boolean>(false);
  const [showRemoveMemberModal, setShowRemoveMemberModal] = useState<boolean>(
    false
  );
  const [showEditUserModal, setShowEditUserModal] = useState<boolean>(false);
  const [showCreateUserModal, setShowCreateUserModal] = useState<boolean>(
    false
  );
  const [isFormSubmitting, setIsFormSubmitting] = useState<boolean>(false);
  const [userEditing, setUserEditing] = useState<IUser>();
  const [searchString, setSearchString] = useState<string>("");
  const [createUserErrors, setCreateUserErrors] = useState<any>(
    DEFAULT_CREATE_USER_ERRORS
  );
  const [editUserErrors, setEditUserErrors] = useState<any>(
    DEFAULT_CREATE_USER_ERRORS
  );
  const [members, setMembers] = useState<IMembersTableData[]>([]);
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
    data: users,
    isLoading: isLoadingUsers,
    error: loadingUsersError,
    refetch: refetchUsers,
  } = useQuery<IUser[], Error, IMembersTableData[]>(
    ["users", teamId, searchString],
    () => usersAPI.loadAll({ teamId, globalFilter: searchString }),
    {
      select: (data: IUser[]) => generateDataSet(teamId, data),
      onSuccess: (data) => {
        setMembers(data);
        setMemberIds(data.map((member) => member.id));
      },
    }
  );

  const {
    data: teams,
    isLoading: isLoadingTeams,
    error: loadingTeamsError,
  } = useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams", teamId],
    () => teamsAPI.loadAll(),
    {
      select: (data: ITeamsResponse) => data.teams,
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
    currentUser ? setUserEditing(currentUser) : setUserEditing(undefined);
  }, [
    showCreateUserModal,
    currentUser,
    setShowCreateUserModal,
    setUserEditing,
    setShowAddMemberModal,
  ]);

  // FUNCTIONS

  const onRemoveMemberSubmit = useCallback(() => {
    const removedUsers = { users: [{ id: userEditing?.id }] };
    teamsAPI
      .removeMembers(teamId, removedUsers)
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully removed ${userEditing?.name}`)
        );
        // If user removes self from team,
        // redirect to home
        if (currentUser && currentUser.id === removedUsers.users[0].id) {
          window.location.href = "/";
        }
      })
      .catch(() =>
        dispatch(
          renderFlash("error", "Unable to remove members. Please try again.")
        )
      )
      .finally(() => {
        toggleRemoveMemberModal();
        refetchUsers();
      });
  }, [
    dispatch,
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
          dispatch(
            renderFlash(
              "success",
              `${newMembers.users.length} members successfully added to ${currentTeam?.name}.`
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not add members. Please try again.")
          );
        })
        .finally(() => {
          toggleAddUserModal();
          refetchUsers();
        });
    },
    [dispatch, teamId, toggleAddUserModal, currentTeam?.name, refetchUsers]
  );

  const fetchUsers = useCallback(
    (fetchParams: IFetchParams) => {
      const { pageIndex, pageSize, searchQuery } = fetchParams;
      usersAPI.loadAll({
        page: pageIndex,
        perPage: pageSize,
        globalFilter: searchQuery,
        teamId,
      });
    },
    [dispatch, teamId]
  );

  const onCreateMemberSubmit = (formData: IFormData) => {
    setIsFormSubmitting(true);

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
          dispatch(
            renderFlash(
              "success",
              `An invitation email was sent from ${config?.sender_address} to ${formData.email}.`
            )
          );
          fetchUsers(tableQueryData);
          toggleCreateMemberModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (
            userErrors.data.errors[0].reason.includes(
              "a user with this account already exists"
            )
          ) {
            setCreateUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("Invite") &&
            userErrors.data.errors[0].reason.includes("already exists")
          ) {
            setCreateUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else {
            dispatch(
              renderFlash("error", "Could not invite user. Please try again.")
            );
          }
        })
        .finally(() => {
          setIsFormSubmitting(false);
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
          dispatch(
            renderFlash("success", `Successfully created ${requestData.name}.`)
          );
          fetchUsers(tableQueryData);
          toggleCreateMemberModal();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("Duplicate")) {
            setCreateUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("already invited")
          ) {
            setCreateUserErrors({
              email: "A user with this email address has already been invited",
            });
          } else {
            dispatch(
              renderFlash("error", "Could not create user. Please try again.")
            );
          }
        })
        .finally(() => {
          setIsFormSubmitting(false);
        });
    }
  };

  const onEditMemberSubmit = useCallback(
    (formData: IFormData) => {
      const updatedAttrs = userManagementHelpers.generateUpdateData(
        userEditing as IUser,
        formData
      );

      setIsFormSubmitting(true);

      const userName = userEditing?.name;

      userEditing &&
        usersAPI
          .update(userEditing.id, updatedAttrs)
          .then(() => {
            dispatch(
              renderFlash("success", `Successfully edited ${userName}.`)
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
              refetchUsers(tableQueryData);
            }
            setIsFormSubmitting(false);
            toggleEditMemberModal();
          })
          .catch((userErrors: { data: IApiError }) => {
            if (userErrors.data.errors[0].reason.includes("already exists")) {
              setEditUserErrors({
                email: "A user with this email address already exists",
              });
            } else {
              dispatch(
                renderFlash(
                  "error",
                  `Could not edit ${userName}. Please try again.`
                )
              );
            }
          });
    },
    [dispatch, toggleEditMemberModal, userEditing, refetchUsers]
  );

  const onQueryChange = useCallback(
    (queryData) => {
      if (members) {
        setSearchString(queryData.searchQuery);
        tableQueryData = { ...queryData, teamId };
        refetchUsers(queryData);
      }
    },
    [refetchUsers, teamId, setSearchString]
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

  const NoMembersComponent = useCallback(() => {
    return (
      <div className={`${noMembersClass}`}>
        <div className={`${noMembersClass}__inner`}>
          <div className={`${noMembersClass}__inner-text`}>
            {searchString === "" ? (
              <>
                <h1>This team doesn&apos;t have any members yet.</h1>
                <p>
                  Expecting to see new team members listed here? Try again in a
                  few seconds as the system catches up.
                </p>
                <Button
                  variant="brand"
                  className={`${noMembersClass}__create-button`}
                  onClick={toggleAddUserModal}
                >
                  Add member
                </Button>
              </>
            ) : (
              <>
                <h2>We couldn’t find any members.</h2>
                <p>
                  Expecting to see members? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString, toggleAddUserModal]);

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
      {loadingUsersError ||
      loadingTeamsError ||
      (!currentTeam && !isLoadingTeams && !isLoadingUsers) ? (
        <TableDataError />
      ) : (
        <TableContainer
          resultsTitle={"members"}
          columns={tableHeaders}
          data={members}
          isLoading={isLoadingUsers}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          onActionButtonClick={toggleAddUserModal}
          actionButtonText={"Add member"}
          actionButtonVariant={"brand"}
          hideActionButton={memberIds.length === 0 && searchString === ""}
          onQueryChange={onQueryChange}
          inputPlaceHolder={"Search"}
          emptyComponent={NoMembersComponent}
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
      {showEditUserModal ? (
        <EditUserModal
          editUserErrors={editUserErrors}
          onCancel={toggleEditMemberModal}
          onSubmit={onEditMemberSubmit}
          defaultName={userEditing?.name}
          defaultEmail={userEditing?.email}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeamRole={userEditing?.role}
          defaultTeams={userEditing?.teams}
          availableTeams={teams || []}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          isSsoEnabled={userEditing?.sso_enabled}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          currentTeam={currentTeam}
        />
      ) : null}
      {showCreateUserModal ? (
        <CreateUserModal
          createUserErrors={createUserErrors}
          onCancel={toggleCreateMemberModal}
          onSubmit={onCreateMemberSubmit}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeamRole={userEditing?.role}
          defaultTeams={userEditing?.teams}
          availableTeams={teams}
          isPremiumTier={isPremiumTier || false}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          currentTeam={currentTeam}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          isFormSubmitting={isFormSubmitting}
        />
      ) : null}
      {showRemoveMemberModal && currentTeam ? (
        <RemoveMemberModal
          memberName={userEditing?.name || ""}
          teamName={currentTeam.name}
          onCancel={toggleRemoveMemberModal}
          onSubmit={onRemoveMemberSubmit}
        />
      ) : null}
    </div>
  );
};

export default MembersPage;
