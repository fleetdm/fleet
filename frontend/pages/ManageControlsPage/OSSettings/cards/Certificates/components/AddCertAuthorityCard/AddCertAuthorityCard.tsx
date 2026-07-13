import React from "react";

import PATHS from "router/paths";
import { InjectedRouter } from "react-router";

import Card from "components/Card";
import Button from "components/buttons/Button";

const baseClass = "add-cert-authority-card";

interface IAddCertAuthorityCardProps {
  router: InjectedRouter;
}

const AddCertAuthorityCard = ({ router }: IAddCertAuthorityCardProps) => (
  <Card className={baseClass}>
    <div className={`${baseClass}__content-wrap`}>
      <div className={`${baseClass}__text`}>
        <b>Add certificate authority</b>
        <p>
          To add certificates, a custom SCEP certificate authority must be
          configured in organization settings.
        </p>
      </div>
      <Button
        className={`${baseClass}__add-button`}
        type="button"
        onClick={() =>
          router.push(PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES)
        }
      >
        Add CA
      </Button>
    </div>
  </Card>
);

export default AddCertAuthorityCard;
