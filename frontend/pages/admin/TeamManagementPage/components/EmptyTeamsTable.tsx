import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";

const EmptyTeamsTable = ({
  className,
  onActionButtonClick,
  disabledPrimaryActionTooltip,
}: {
  className: string;
  onActionButtonClick: () => void;
  // covers both disabling teams UI for Primo and GitOps mode, with correct precedence
  disabledPrimaryActionTooltip: React.ReactNode;
}) => {
  const rawButton = (
    <Button
      disabled={!!disabledPrimaryActionTooltip}
      onClick={onActionButtonClick}
      className={`${className}__create-button`}
    >
      Create team
    </Button>
  );
  const primaryButton = disabledPrimaryActionTooltip ? (
    <TooltipWrapper
      tipContent={disabledPrimaryActionTooltip}
      position="top"
      underline={false}
      showArrow
      tipOffset={8}
    >
      {rawButton}
    </TooltipWrapper>
  ) : (
    rawButton
  );

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
      primaryButton={primaryButton}
    />
  );
};

export default EmptyTeamsTable;
