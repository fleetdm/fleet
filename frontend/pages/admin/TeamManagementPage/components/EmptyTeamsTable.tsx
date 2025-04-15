import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

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
        <GitOpsModeTooltipWrapper
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <Button
              className={`${className}__create-button`}
              onClick={onActionButtonClick}
              disabled={disableChildren}
            >
              Create team
            </Button>
          )}
        />
      }
    />
  );
};

export default EmptyTeamsTable;
