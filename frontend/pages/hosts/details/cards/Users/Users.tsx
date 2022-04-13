import React from "react";

import { IHostUser } from "interfaces/host_users";
import TableContainer from "components/TableContainer";

import generateUsersTableHeaders from "./UsersTable/UsersTableConfig";
import EmptyUsers from "./EmptyUsers";

interface ISearchQueryData {
  searchQuery: string;
  sortHeader: string;
  sortDirection: string;
  pageSize: number;
  pageIndex: number;
}

interface IUsersProps {
  users: IHostUser[];
  usersState: { username: string }[];
  isLoading: boolean;
  onUsersTableSearchChange: (queryData: ISearchQueryData) => void;
}

const Users = ({
  users,
  usersState,
  isLoading,
  onUsersTableSearchChange,
}: IUsersProps): JSX.Element => {
  const tableHeaders = generateUsersTableHeaders();

  if (users) {
    return (
      <div className="section section--users">
        <p className="section__header">Users</p>
        {users.length === 0 ? (
          <p className="results__data">No users were detected on this host.</p>
        ) : (
          <TableContainer
            columns={tableHeaders}
            data={usersState}
            isLoading={isLoading}
            defaultSortHeader={"username"}
            defaultSortDirection={"asc"}
            inputPlaceHolder={"Search users by username"}
            onQueryChange={onUsersTableSearchChange}
            resultsTitle={"users"}
            emptyComponent={EmptyUsers}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            searchable
            wideSearch
            filteredCount={usersState.length}
            isClientSidePagination
          />
        )}
      </div>
    );
  }

  return (
    <div className="section section--users">
      <p className="section__header">Users</p>
      <p className="results__data">No users were detected on this host.</p>
    </div>
  );
};

export default Users;
