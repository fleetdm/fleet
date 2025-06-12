import React from "react";

import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button/Button";

const baseClass = "require-end-user-auth";

interface IRequireEndUserAuthProps {
  onClickConnect: () => void;
}

const RequireEndUserAuth = ({ onClickConnect }: IRequireEndUserAuthProps) => {
  return (
    <div className={baseClass}>
      <EmptyTable
        header="Require end user authentication during setup"
        info="Connect Fleet to your identity provider (IdP) to get started."
        primaryButton={<Button onClick={onClickConnect}>Connect</Button>}
        className={`${baseClass}__required-message`}
      />
    </div>
  );
};

export default RequireEndUserAuth;
