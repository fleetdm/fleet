import React from "react";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

interface IAddCertCardProps {
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddCertCard = ({ setShowModal }: IAddCertCardProps) => (
  <EmptyState
    variant="header-list"
    header="No certificates"
    info="Add a certificate to help your end users connect to your corporate network."
    primaryButton={
      <GitOpsModeTooltipWrapper
        tipOffset={8}
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            type="button"
            onClick={() => setShowModal(true)}
          >
            Add certificate
          </Button>
        )}
      />
    }
  />
);

export default AddCertCard;
