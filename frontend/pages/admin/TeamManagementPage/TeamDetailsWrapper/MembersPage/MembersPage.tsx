import React, { useCallback, useContext, useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
// @ts-ignore
import memoize from "memoize-one";
import PATHS from "router/paths";
import { IConfig } from "interfaces/config";
import { IUser } from "interfaces/user";
import { INewMembersBody, ITeam } from "interfaces/team";
import { Link } from "react-router";
import { AppContext } from "context/app";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import userActions from "redux/nodes/entities/users/actions";
import teamActions from "redux/nodes/entities/teams/actions";
// @ts-ignore
import inviteActions from "redux/nodes/entities/invites/actions";
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
} from "./MembersPageTableConfig";

const baseClass = "members";
const noMembersClass = "no-members";

interface IMembersPageProps {
  params: {
    team_id: string;
  };
}

interface IRootState {
  app: {
    config: IConfig;
  };
  entities: {
    users: {
      loading: boolean;
      data: { [id: number]: IUser };
      errors: { name: string; reason: string }[];
    };
    teams: {
      data: { [id: number]: ITeam };
    };
  };
}

interface IFetchParams {
  pageIndex?: number;
  pageSize?: number;
  searchQuery?: string;
}

const getTeams = (data: { [id: string]: ITeam }) => {
  return Object.keys(data).map((teamId) => {
    return data[teamId];
  });
};

const memoizedGetTeams = memoize(getTeams);

// This is used to cache the table query data and make a request for the
// members data at a future time. Practically, this allows us to re-fetch the users
// with the same table query params after we have made an edit to a user.
let tableQueryData = {};

const MembersPage = ({
  params: { team_id },
}: IMembersPageProps): JSX.Element => {
  const teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();

  const { isGlobalAdmin, currentUser } = useContext(AppContext);

  const isPremiumTier = useSelector((state: IRootState) => {
    return state.app.config.tier === "premium";
  });
  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.users.loading
  );

  const users = useSelector((state: IRootState) =>
    generateDataSet(teamId, state.entities.users.data)
  );

  const usersError = useSelector(
    (state: IRootState) => state.entities.users.errors
  );

  const team = useSelector((state: IRootState) => {
    return state.entities.teams.data[teamId];
  });
  const teams = useSelector((state: IRootState) => {
    return memoizedGetTeams(state.entities.teams.data);
  });
  const memberIds = users.map((member) => {
    return member.id;
  });

  const smtpConfigured = useSelector((state: IRootState) => {
    return state.app.config.configured;
  });

  const canUseSso = useSelector((state: IRootState) => {
    return state.app.config.enable_sso;
  });

  const config = useSelector((state: IRootState) => {
    return state.app.config;
  });

  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [showRemoveMemberModal, setShowRemoveMemberModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [showCreateUserModal, setShowCreateUserModal] = useState(false);
  const [isFormSubmitting, setIsFormSubmitting] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();
  const [searchString, setSearchString] = useState<string>("");
  const [createUserErrors] = useState(DEFAULT_CREATE_USER_ERRORS);
  const [editUserErrors] = useState(DEFAULT_CREATE_USER_ERRORS);

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

  const toggleEditMemberModal = useCallback(
    (user?: IUser) => {
      setShowEditUserModal(!showEditUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
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

  const onRemoveMemberSubmit = useCallback(() => {
    const removedUsers = { users: [{ id: userEditing?.id }] };
    dispatch(teamActions.removeMembers(teamId, removedUsers))
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
      );
    toggleRemoveMemberModal();
  }, [
    dispatch,
    teamId,
    userEditing?.id,
    userEditing?.name,
    toggleRemoveMemberModal,
  ]);

  const onAddMemberSubmit = useCallback(
    (newMembers: INewMembersBody) => {
      dispatch(teamActions.addMembers(teamId, newMembers))
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `${newMembers.users.length} members successfully added to ${team.name}.`
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not add members. Please try again.")
          );
        });
      toggleAddUserModal();
    },
    [dispatch, teamId, toggleAddUserModal, team.name]
  );

  const fetchUsers = useCallback(
    (fetchParams: IFetchParams) => {
      const { pageIndex, pageSize, searchQuery } = fetchParams;
      dispatch(
        userActions.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
          teamId,
        })
      );
    },
    [dispatch, teamId]
  );

  const onCreateMemberSubmit = (formData: IFormData) => {
    setIsFormSubmitting(true);

    if (formData.newUserType === NewUserType.AdminInvited) {
      // Do some data formatting adding `invited_by` for the request to be correct and deleteing uncessary fields
      const requestData = {
        ...formData,
        invited_by: formData.currentUserId,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      delete requestData.password; // this field is not needed for the request
      dispatch(inviteActions.create(requestData))
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `An invitation email was sent from ${config.sender_address} to ${formData.email}.`
            )
          );
          fetchUsers(tableQueryData);
          toggleCreateMemberModal();
        })
        .catch((userErrors: any) => {
          if (userErrors.base.includes("Duplicate")) {
            dispatch(
              renderFlash(
                "error",
                "A user with this email address already exists."
              )
            );
          } else {
            dispatch(
              renderFlash("error", "Could not create user. Please try again.")
            );
          }
        })
        .finally(() => {
          setIsFormSubmitting(false);
        });
    } else {
      // Do some data formatting deleteing uncessary fields
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      dispatch(userActions.createUserWithoutInvitation(requestData))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully created ${requestData.name}.`)
          );
          fetchUsers(tableQueryData);
          toggleCreateMemberModal();
        })
        .catch((userErrors: any) => {
          if (userErrors.base.includes("Duplicate")) {
            dispatch(
              renderFlash(
                "error",
                "A user with this email address already exists."
              )
            );
          } else if (userErrors.base.includes("already invited")) {
            dispatch(
              renderFlash(
                "error",
                "A user with this email address has already been invited."
              )
            );
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
      dispatch(userActions.update(userEditing, updatedAttrs))
        .then(() => {
          dispatch(renderFlash("success", `Successfully edited ${userName}.`));
          if (currentUser && userEditing && currentUser.id === userEditing.id) {
            // If user edits self and removes "admin" role,
            // redirect to home
            const currentTeam = formData.teams.filter(
              (thisTeam) => thisTeam.id === teamId
            );
            if (currentTeam && currentTeam[0].role !== "admin") {
              window.location.href = "/";
            }
          } else {
            fetchUsers(tableQueryData);
          }
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not edit ${userName}. Please try again.`
            )
          );
        })
        .finally(() => {
          setIsFormSubmitting(false);
        });
      toggleEditMemberModal();
    },
    [dispatch, toggleEditMemberModal, userEditing, fetchUsers]
  );

  useEffect(() => {
    fetchUsers(tableQueryData);
  }, [team_id]);

  // NOTE: this will fire on initial render, so we use this to get the list of
  // users for this team, as well as use it as a handler when the table query
  // changes.
  const onQueryChange = useCallback(
    (queryData) => {
      setSearchString(queryData.searchQuery);
      tableQueryData = { ...queryData, teamId };
      fetchUsers(queryData);
    },
    [fetchUsers, teamId, setSearchString]
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
      {Object.keys(usersError).length > 0 ? (
        <TableDataError />
      ) : (
        <TableContainer
          resultsTitle={"members"}
          columns={tableHeaders}
          data={users}
          isLoading={loadingTableData}
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
          isClientSideSearch
        />
      )}
      {showAddMemberModal ? (
        <AddMemberModal
          team={team}
          disabledMembers={memberIds}
          onCancel={toggleAddUserModal}
          onSubmit={onAddMemberSubmit}
          onCreateNewMember={toggleCreateMemberModal}
        />
      ) : null}
      {showEditUserModal ? (
        <EditUserModal
          serverErrors={editUserErrors}
          onCancel={toggleEditMemberModal}
          onSubmit={onEditMemberSubmit}
          defaultName={userEditing?.name}
          defaultEmail={userEditing?.email}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeamRole={userEditing?.role}
          defaultTeams={userEditing?.teams}
          availableTeams={teams}
          isPremiumTier={isPremiumTier}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          isSsoEnabled={userEditing?.sso_enabled}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          currentTeam={team}
        />
      ) : null}
      {showCreateUserModal ? (
        <CreateUserModal
          serverErrors={createUserErrors}
          onCancel={toggleCreateMemberModal}
          onSubmit={onCreateMemberSubmit}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeamRole={userEditing?.role}
          defaultTeams={userEditing?.teams}
          availableTeams={teams}
          isPremiumTier={isPremiumTier}
          smtpConfigured={smtpConfigured}
          canUseSso={canUseSso}
          currentTeam={team}
          isModifiedByGlobalAdmin={isGlobalAdmin}
          isFormSubmitting={isFormSubmitting}
        />
      ) : null}
      {showRemoveMemberModal ? (
        <RemoveMemberModal
          memberName={userEditing?.name || ""}
          teamName={team.name}
          onCancel={toggleRemoveMemberModal}
          onSubmit={onRemoveMemberSubmit}
        />
      ) : null}
    </div>
  );
};

export default MembersPage;
