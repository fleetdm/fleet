import React from "react";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

interface IAddCertAuthoityCardProps {
  onAddCertAuthority: () => void;
}

const AddCertAuthorityCard = ({
  onAddCertAuthority,
}: IAddCertAuthoityCardProps) => {
  return (
    <EmptyState
      variant="header-list"
      header="Add your certificate authority (CA)"
      info="Help your end users connect to Wi-Fi or VPNs."
      primaryButton={
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button disabled={disableChildren} onClick={onAddCertAuthority}>
              Add CA
            </Button>
          )}
        />
      }
    />
  );
};

export default AddCertAuthorityCard;
