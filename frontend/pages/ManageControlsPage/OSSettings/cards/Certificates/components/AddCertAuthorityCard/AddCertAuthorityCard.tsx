import React from "react";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import PATHS from "router/paths";

interface IAddCertAuthorityCardProps {
  router: InjectedRouter;
}

const AddCertAuthorityCard = ({ router }: IAddCertAuthorityCardProps) => (
  <EmptyState
    variant="list"
    header="No certificate authority"
    info="Add a custom SCEP certificate authority to deploy certificates to your end users."
    primaryButton={
      <Button
        type="button"
        onClick={() =>
          router.push(PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES)
        }
      >
        Add CA
      </Button>
    }
  />
);

export default AddCertAuthorityCard;
