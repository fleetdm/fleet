import React, { useCallback, useState } from "react";

import { IUser } from "interfaces/user";
import TableContainer from "components/TableContainer";
import EditUserModal from "../../../UserManagementPage/components/EditUserModal";
import CreateTeamModal from "../../components/CreateTeamModal";
import DeleteTeamModal from "../../components/DeleteTeamModal";

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

  const toggleEditUserModal = useCallback(
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
        toggleEditUserModal(user);
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
      username: "test username",
      sso_enabled: false,
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
      id: 2,
      name: "Test 2",
      username: "test 2 username",
      sso_enabled: false,
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
