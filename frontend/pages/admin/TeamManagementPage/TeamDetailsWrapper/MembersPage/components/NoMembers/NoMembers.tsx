import React from "react";

const baseClass = "no-members";

const NoMembers = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <h2>Rally the fleet</h2>
      <p>Add your first team members and start organizing their permissions.</p>
    </div>
  );
};

export default NoMembers;
