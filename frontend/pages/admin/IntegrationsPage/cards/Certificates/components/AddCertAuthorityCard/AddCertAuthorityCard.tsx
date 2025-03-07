import React from "react";

import Button from "components/buttons/Button";
import Card from "components/Card";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "add-cert-authority-card";

interface IAddCertAuthoityCardProps {
  onAddCertAuthority: () => void;
}

const AddCertAuthorityCard = ({
  onAddCertAuthority,
}: IAddCertAuthoityCardProps) => {
  return (
    <Card paddingSize="xxlarge" className={baseClass}>
      <div className={`${baseClass}__content`}>
        <p className={`${baseClass}__title`}>
          Add your certificate authority (CA)
        </p>
        <p>Help your end users connect to Wi-Fi or VPNs.</p>
      </div>
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            disabled={disableChildren}
            className={`${baseClass}__button`}
            onClick={onAddCertAuthority}
          >
            Add CA
          </Button>
        )}
      />
    </Card>
  );
};

export default AddCertAuthorityCard;
