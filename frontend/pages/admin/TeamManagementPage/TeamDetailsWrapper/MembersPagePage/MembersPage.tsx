import React, { useCallback, useEffect, useState } from "react";

import { IUser } from "interfaces/user";
import { INewMembersBody } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import userActions from "redux/nodes/entities/users/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import TableContainer from "components/TableContainer";
import { useDispatch, useSelector } from "react-redux";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import AddMemberModal from "./components/AddMemberModal";
import EmptyMembers from "./components/EmptyMembers";

import {
  generateTableHeaders,
  generateDataSet,
} from "./MembersPageTableConfig";

const baseClass = "members";

interface IMembersPageProps {
  params: {
    team_id: number;
  };
}

interface IRootState {
  entities: {
    users: {
      loading: boolean;
      data: { [id: number]: IUser };
    };
  };
}

const MembersPage = (props: IMembersPageProps): JSX.Element => {
  const {
    params: { team_id },
  } = props;
  const dispatch = useDispatch();

  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [showRemoveUserModal, setShowRemoveUserModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();

  useEffect(() => {
    dispatch(userActions.loadAll({ teamId: team_id }));
  }, [dispatch, team_id]);

  const toggleAddUserModal = useCallback(() => {
    setShowAddMemberModal(!showAddMemberModal);
  }, [showAddMemberModal, setShowAddMemberModal]);

  const toggleRemoveUserModal = useCallback(
    (user?: IUser) => {
      setShowRemoveUserModal(!showRemoveUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
    },
    [showRemoveUserModal, setShowRemoveUserModal, setUserEditing]
  );

  const toggleEditUserModal = useCallback(
    (user?: IUser) => {
      setShowEditUserModal(!showEditUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );

  const onAddMemberSubmit = useCallback(
    (newMembers: INewMembersBody) => {
      dispatch(teamActions.addMembers(team_id, newMembers)).then(() => {
        dispatch(
          renderFlash(
            "success",
            `${newMembers.users.length} members successfully added to TEAM.`
          )
        ); // TODO: update team name
      });
      setShowAddMemberModal(false);
    },
    [dispatch, team_id]
  );

  const onQueryChange = useCallback(
    (queryData) => {
      const { pageIndex, pageSize, searchQuery } = queryData;
      dispatch(
        userActions.loadAll({
          page: pageIndex,
          perPage: pageSize,
          globalFilter: searchQuery,
          teamId: team_id,
        })
      );
    },
    [dispatch, team_id]
  );

  // NOTE: we are purposely showing edit modal.
  const onActionSelection = (action: string, user: IUser): void => {
    switch (action) {
      case "edit":
        toggleEditUserModal(user);
        break;
      case "remove":
        toggleEditUserModal(user);
        break;
      default:
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);

  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.users.loading
  );
  const users = useSelector((state: IRootState) =>
    generateDataSet(state.entities.users.data)
  );

  return (
    <div className={baseClass}>
      <p>Add, customize, and remove members from Walmart Pay.</p>
      <h2>Members Page</h2>
      <TableContainer
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
          onCancel={toggleEditUserModal}
          onSubmit={() => console.log("submit")}
          defaultName={userEditing?.name}
          defaultEmail={userEditing?.email}
          defaultGlobalRole={userEditing?.global_role}
          defaultTeams={userEditing?.teams}
          defaultSSOEnabled={userEditing?.sso_enabled}
          availableTeams={[]}
          validationErrors={[]}
        />
      ) : null}
    </div>
  );
};

export default MembersPage;
