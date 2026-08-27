import React from "react";

import Button from "components/buttons/Button";
import SectionCard from "../../SectionCard";

interface IWindowsAutomaticEnrollmentCardProps {
  windowsMdmEnabled: boolean;
  tenantAdded: boolean;
  onViewDetails: () => void;
}

const WindowsMdmDisabledCard = (
  <SectionCard header="Windows enrollment">
    To enable end users to enroll to Fleet via Microsoft Entra (e.g. Autopilot),
    first turn on Windows MDM.
  </SectionCard>
);

interface IWindowsTenantAddedCardProps {
  editTenants: () => void;
}

const WindowsTenantAddedCard = ({
  editTenants,
}: IWindowsTenantAddedCardProps) => (
  <SectionCard
    iconName="success"
    cta={
      <Button onClick={editTenants} variant="subdued" icon="pencil">
        Edit
      </Button>
    }
  >
    Microsoft Entra tenant ID added.
  </SectionCard>
);

interface IWindowsTenantNotAddedCardProps {
  addTenant: () => void;
}

const WindowsTenantNotAddedCard = ({
  addTenant,
}: IWindowsTenantNotAddedCardProps) => (
  <SectionCard
    header="Windows enrollment"
    cta={<Button onClick={addTenant}>Connect</Button>}
  >
    To enable end users to enroll to Fleet via Microsoft Entra (e.g. Autopilot),
    you need to add Entra tenant ID first.
  </SectionCard>
);

const WindowsAutomaticEnrollmentCard = ({
  windowsMdmEnabled,
  tenantAdded,
  onViewDetails,
}: IWindowsAutomaticEnrollmentCardProps) => {
  if (!windowsMdmEnabled) {
    return WindowsMdmDisabledCard;
  }

  if (tenantAdded) {
    return <WindowsTenantAddedCard editTenants={onViewDetails} />;
  }

  return <WindowsTenantNotAddedCard addTenant={onViewDetails} />;
};

export default WindowsAutomaticEnrollmentCard;
