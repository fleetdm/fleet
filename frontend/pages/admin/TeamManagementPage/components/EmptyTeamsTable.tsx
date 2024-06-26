import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";

const EmptyTeamsTable = ({
  className,
  onActionButtonClick,
}: {
  className: string;
  onActionButtonClick: () => void;
}) => {
  return (
    <EmptyTable
      graphicName="empty-teams"
      header="Set up team permissions"
      info="Keep your organization organized and efficient by ensuring every user has the correct access to the right hosts."
      additionalInfo={
        <>
          {" "}
          Want to learn more?&nbsp;
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/teams"
            text="Read about teams"
            newTab
          />
        </>
      }
      primaryButton={
        <Button
          variant="brand"
          className={`${className}__create-button`}
          onClick={onActionButtonClick}
        >
          Create team
        </Button>
      }
    />
  );
};

export default EmptyTeamsTable;
