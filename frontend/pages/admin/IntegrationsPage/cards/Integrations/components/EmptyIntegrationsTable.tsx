import React from "react";

import Button from "components/buttons/Button";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const EmptyIntegrationsTable = ({
  className,
  onActionButtonClick,
}: {
  className: string;
  onActionButtonClick: () => void;
}) => {
  return (
    <EmptyTable
      graphicName="empty-integrations"
      header="Ticket destinations"
      info="Create tickets automatically when Fleet detects new software vulnerabilities or hosts failing policies."
      additionalInfo={
        <>
          Want to learn more?&nbsp;
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/automations"
            text="Read about automations"
            newTab
          />
        </>
      }
      primaryButton={
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              className={`${className}__add-button`}
              onClick={onActionButtonClick}
              disabled={disableChildren}
            >
              Add
            </Button>
          )}
        />
      }
    />
  );
};

export default EmptyIntegrationsTable;
