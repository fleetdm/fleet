import React, { useCallback, useState } from "react";

import { IUser } from "interfaces/user";
import TableContainer from "components/TableContainer";
import CreateTeamModal from "../../components/CreateTeamModal";
import DeleteTeamModal from "../../components/DeleteTeamModal";
import EditTeamModal from "../../components/EditTeamModal";

import {
  generateTableHeaders,
  generateDataSet,
} from "./MembersPageTableConfig";

const baseClass = "members";

const MembersPage = (): JSX.Element => {
  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [showRemoveUserModal, setShowRemoveUserModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [userEditing, setUserEditing] = useState<IUser>();

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

  const toggleEditTeamModal = useCallback(
    (user?: IUser) => {
      setShowEditUserModal(!showEditUserModal);
      user ? setUserEditing(user) : setUserEditing(undefined);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );

  // NOTE: called once on the initial render of this component.
  // const onQueryChange = useCallback(
  //   (queryData) => {
  //     const { pageIndex, pageSize, searchQuery } = queryData;
  //     dispatch(teamActions.loadAll(pageIndex, pageSize, searchQuery));
  //   },
  //   [dispatch]
  // );

  const onQueryChange = useCallback(() => {
    console.log("query change");
  }, []);

  const onActionSelection = (action: string, user: IUser): void => {
    switch (action) {
      case "edit":
        toggleEditTeamModal(user);
        break;
      case "remove":
        toggleRemoveUserModal(user);
        break;
      default:
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const tableData = generateDataSet({
    1: {
      admin: false,
      email: "test+1@fleetdm.com",
      enabled: true,
      force_password_reset: false,
      gravatarURL: "test",
      id: 1,
      name: "Test 1",
      position: "test position",
      username: "test username",
      teams: [
        {
          name: "Test Team 1",
          id: 1,
          hosts: 10,
          members: 10,
          role: "Member",
        },
      ],
      global_role: null,
    },
    2: {
      admin: false,
      email: "test+2@fleetdm.com",
      enabled: true,
      force_password_reset: false,
      gravatarURL: "test",
      id: 1,
      name: "Test 2",
      position: "test 2 position",
      username: "test 2 username",
      teams: [
        {
          name: "Test Team 2",
          id: 1,
          hosts: 10,
          members: 10,
          role: "Observer",
        },
      ],
      global_role: null,
    },
  });

  return (
    <div className={baseClass}>
      <p>Add, customize, and remove members from Walmart Pay.</p>
      <h2>Members Page</h2>
      <TableContainer
        columns={tableHeaders}
        data={tableData}
        isLoading={false}
        defaultSortHeader={"name"}
        defaultSortDirection={"asc"}
        onActionButtonClick={toggleAddUserModal}
        actionButtonText={"Add member"}
        onQueryChange={onQueryChange}
        inputPlaceHolder={"Search"}
        emptyComponent={() => <p>Empty Members</p>}
      />
      {showAddMemberModal ? (
        <CreateTeamModal
          onCancel={toggleAddUserModal}
          // onSubmit={onCreateSubmit}
          onSubmit={() => console.log("add submit")}
        />
      ) : null}
      {showRemoveUserModal ? (
        <DeleteTeamModal
          onCancel={toggleRemoveUserModal}
          // onSubmit={onDeleteSubmit}
          onSubmit={() => console.log("remove submit")}
          name={userEditing?.name || ""}
        />
      ) : null}
      {showEditUserModal ? (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          // onSubmit={onEditSubmit}
          onSubmit={() => console.log("edit submit")}
          defaultName={userEditing?.name || ""}
        />
      ) : null}
    </div>
  );
};

export default MembersPage;
