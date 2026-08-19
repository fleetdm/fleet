import React from "react";
import { InjectedRouter } from "react-router";

import PageDescription from "components/PageDescription";
import UsersTable from "./components/UsersTable";

const baseClass = "manage-users";

interface IUserManagementProps {
  router: InjectedRouter; // v3
}

const ManageUsersPage = ({ router }: IUserManagementProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <PageDescription content="Manage who can access Fleet and what they can do." />
      <UsersTable router={router} />
    </div>
  );
};

export default ManageUsersPage;
