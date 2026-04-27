import React from "react";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import CustomLink from "components/CustomLink";

const EmptyIntegrationsTable = ({
  className,
  onActionButtonClick,
}: {
  className: string;
  onActionButtonClick: () => void;
}) => {
  return (
    <EmptyState
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
        <Button
          className={`${className}__add-button`}
          onClick={onActionButtonClick}
        >
          Add
        </Button>
      }
    />
  );
};

export default EmptyIntegrationsTable;
