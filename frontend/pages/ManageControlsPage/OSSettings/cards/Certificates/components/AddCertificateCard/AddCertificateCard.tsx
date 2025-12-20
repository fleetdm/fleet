import React from "react";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Card from "components/Card";
import Button from "components/buttons/Button";

const baseClass = "create-cert-card";

interface ICreateCertCardProps {
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const CreateCertCard = ({ setShowModal }: ICreateCertCardProps) => (
  <Card className={baseClass}>
    <div className={`${baseClass}__content-wrap`}>
      <div className={`${baseClass}__text`}>
        <b>Create certificate</b>
        <p>Help your end users connect to your corporate network.</p>
      </div>
      <GitOpsModeTooltipWrapper
        tipOffset={8}
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            className={`${baseClass}__card--create-button`}
            type="button"
            onClick={() => setShowModal(true)}
          >
            Create
          </Button>
        )}
      />
    </div>
  </Card>
);

export default CreateCertCard;
