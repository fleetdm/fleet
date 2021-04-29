import React, { useCallback, useEffect, useState } from "react";
// @ts-ignore
import memoize from "memoize-one";

import { IUser } from "interfaces/user";
import { INewMembersBody, ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import userActions from "redux/nodes/entities/users/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import TableContainer from "components/TableContainer";
import { useDispatch, useSelector } from "react-redux";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import { IFormData } from "../../../UserManagementPage/components/UserForm/UserForm";
import userManagementHelpers from "../../../UserManagementPage/helpers";
import AddMemberModal from "./components/AddMemberModal";
import EmptyMembers from "./components/EmptyMembers";
import RemoveMemberModal from "./components/RemoveMemberModal";

import {
  generateTableHeaders,
  generateDataSet,
} from "./MembersPageTableConfig";

const baseClass = "members";

interface IMembersPageProps {
  params: {
    team_id: string;
  };
}

interface IRootState {
  entities: {
    users: {
      loading: boolean;
      data: { [id: number]: IUser };
    };
    teams: {
      data: { [id: number]: ITeam };
    };
  };
}

const getTeams = (data: { [id: string]: ITeam }) => {
  return Object.keys(data).map((teamId) => {
    return data[teamId];
  });
};

const memoizedGetTeams = memoize(getTeams);

const MembersPage = (props: IMembersPageProps): JSX.Element => {
  const {
    params: { team_id },
  } = props;
  const teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();

  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.users.loading
  );
  const users = useSelector((state: IRootState) =>
    generateDataSet(teamId, state.entities.users.data)
  );
  const team = useSelector((state: IRootState) => {
    return state.entities.teams.data[teamId];
  });
  const teams = useSelector((state: IRootState) => {
    return memoizedGetTeams(state.entities.teams.data);
  });

  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [showRemoveMemberModal, setShowRemoveMemberModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();

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

  const onRemoveMemberSubmit = useCallback(() => {
    const removedUsers = { users: [{ id: userEditing?.id }] };
    dispatch(teamActions.removeMembers(teamId, removedUsers))
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully removed ${userEditing?.name}`)
        );
      })
      .catch(() => dispatch(renderFlash("error", "Remove failed")));
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
          ); // TODO: update team name
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not add members. Please try again.")
          );
        });
      toggleAddUserModal();
    },
    [dispatch, teamId, toggleAddUserModal]
  );

  const onEditMemberSubmit = useCallback(
    (formData: IFormData) => {
      const updatedAttrs = userManagementHelpers.generateUpdateData(
        userEditing as IUser,
        formData
      );

      const userName = userEditing?.name;
      dispatch(userActions.update(userEditing, updatedAttrs))
        .then(() => {
          dispatch(renderFlash("success", `Successfully edited ${userName}.`));
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not edit ${userName}. Please try again.`
            )
          );
        });
      toggleEditMemberModal();
    },
    [dispatch, toggleEditMemberModal, userEditing]
  );

  // NOTE: this will fire on initial render, so we use this to get the list of
  // users for this team, as well as use it as a handler when the table query
  // changes.
  const onQueryChange = useCallback(
    (queryData) => {
      const { pageIndex, pageSize, searchQuery } = queryData;
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

  const tableHeaders = generateTableHeaders(onActionSelection);

  return (
    <div className={baseClass}>
      <p>Add, customize, and remove members from {team.name}.</p>
      <h2>Members Page</h2>
      <TableContainer
        resultsTitle={"members"}
        columns={tableHeaders}
        data={users}
        isLoading={loadingTableData}
        defaultSortHeader={"name"}
        defaultSortDirection={"asc"}
        onActionButtonClick={toggleAddUserModal}
        actionButtonText={"Add member"}
        onQueryChange={onQueryChange}
        inputPlaceHolder={"Search"}
        emptyComponent={EmptyMembers}
      />
      {showAddMemberModal ? (
        <AddMemberModal
          onCancel={toggleAddUserModal}
          onSubmit={onAddMemberSubmit}
        />
      ) : null}
      {showEditUserModal ? (
        <EditUserModal
          onCancel={toggleEditMemberModal}
          onSubmit={onEditMemberSubmit}
          defaultName={userEditing?.name}
          defaultEmail={userEditing?.email}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeams={userEditing?.teams}
          defaultSSOEnabled={userEditing?.sso_enabled}
          availableTeams={teams}
          validationErrors={[]}
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
