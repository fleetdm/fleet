import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";

const EmptyMembersTable = ({
  className,
  isGlobalAdmin,
  isTeamAdmin,
  searchString,
  toggleAddUserModal,
  toggleCreateMemberModal,
}: {
  className: string;
  searchString: string;
  isGlobalAdmin: boolean;
  isTeamAdmin: boolean;
  toggleAddUserModal: () => void;
  toggleCreateMemberModal: () => void;
}) => {
  let graphicName: "empty-members" | undefined;
  let header = "This team doesn't have any members yet.";
  let info =
    "Expecting to see new team members listed here? Try again in a few seconds as the system catches up.";

  if (searchString !== "") {
    graphicName = "empty-members";
    header = "We couldn’t find any members.";
    info =
      "Expecting to see members? Try again in a few seconds as the system catches up.";
  }

  return (
    <EmptyTable
      graphicName={graphicName}
      header={header}
      info={info}
      primaryButton={
        (isGlobalAdmin && (
          <Button
            variant="brand"
            className={`${className}__create-button`}
            onClick={toggleAddUserModal}
          >
            Add member
          </Button>
        )) ||
        (isTeamAdmin && (
          <Button
            variant="brand"
            className={`${className}__create-button`}
            onClick={toggleCreateMemberModal}
          >
            Create user
          </Button>
        )) ||
        undefined
      }
    />
  );
};

export default EmptyMembersTable;
