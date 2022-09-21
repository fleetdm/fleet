import React from "react";
import { InjectedRouter } from "react-router";
import SandboxGate from "components/Sandbox/SandboxGate";
import SandboxDemoMessage from "components/Sandbox/SandboxDemoMessage";
import UsersTable from "./components/UsersTable";

const baseClass = "user-management";

interface IUserManagementProps {
  router: InjectedRouter; // v3
}

const UserManagementPage = ({ router }: IUserManagementProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Create new users, customize user permissions, and remove users from
        Fleet.
      </p>
      <SandboxGate
        fallbackComponent={() => (
          <SandboxDemoMessage
            message="User management is only available in self-managed Fleet"
            utmSource="fleet-ui-users-page"
            className={`${baseClass}__sandbox-demo-message`}
          />
        )}
      >
        <UsersTable router={router} />
      </SandboxGate>
    </div>
  );
};

export default UserManagementPage;
