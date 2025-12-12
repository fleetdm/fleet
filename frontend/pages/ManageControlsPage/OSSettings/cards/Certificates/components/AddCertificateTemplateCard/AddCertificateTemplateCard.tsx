import React from "react";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Card from "components/Card";
import Button from "components/buttons/Button";

const baseClass = "add-cert-card";

interface IAddCTCardProps {
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddCTCard = ({ setShowModal }: IAddCTCardProps) => (
  <Card color="grey" className={baseClass}>
    <div className={`${baseClass}__card--content-wrap`}>
      <b>Add certificate template</b>
      <p>Help your end users connect to your corporate network.</p>
      <GitOpsModeTooltipWrapper
        tipOffset={8}
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            className={`${baseClass}__card--add-button`}
            type="button"
            onClick={() => setShowModal(true)}
          >
            Add
          </Button>
        )}
      />
    </div>
  </Card>
);

export default AddCTCard;
