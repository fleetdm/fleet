import React from "react";

import { IHostUser } from "interfaces/host_users";
import TableContainer from "components/TableContainer";

import generateUsersTableHeaders from "./UsersTable/UsersTableConfig";
// import EmptyUsers from "./EmptyUsers";
import EmptyState from "../EmptyState";

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
  hostUsersEnabled?: boolean;
}

const Users = ({
  users,
  usersState,
  isLoading,
  onUsersTableSearchChange,
  hostUsersEnabled,
}: IUsersProps): JSX.Element => {
  const tableHeaders = generateUsersTableHeaders();

  const EmptyUserSearch = () => (
    <EmptyState title="users" reason="empty-search" />
  );

  if (!hostUsersEnabled) {
    return (
      <div className="section section--users">
        <p className="section__header">Users</p>
        <EmptyState title="users" reason="disabled" />
      </div>
    );
  }

  return (
    <div className="section section--users">
      <p className="section__header">Users</p>
      {users?.length ? (
        <TableContainer
          columns={tableHeaders}
          data={usersState}
          isLoading={isLoading}
          defaultSortHeader={"username"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search users by username"}
          onQueryChange={onUsersTableSearchChange}
          resultsTitle={"users"}
          emptyComponent={EmptyUserSearch}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          searchable
          wideSearch
          filteredCount={usersState.length}
          isClientSidePagination
        />
      ) : (
        <EmptyState title="users" />
      )}
    </div>
  );
};

export default Users;
