import React from "react";
import { InjectedRouter } from "react-router";
import UsersTable from "./components/UsersTable";

const baseClass = "user-management";

interface IUserManagementProps {
  router: InjectedRouter; // v3
}

const UserManagementPage = ({ router }: IUserManagementProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <UsersTable router={router} />
    </div>
  );
};

export default UserManagementPage;
