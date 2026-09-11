import React from "react";

import Button from "components/buttons/Button";
import { IconNames } from "components/icons";
import SectionCard from "../../SectionCard";

interface IMicrosoftGraphCardProps {
  /** Whether a Graph credential is stored. */
  credentialAdded: boolean;
  /** Whether the stored credential was rejected by Entra on the last sync. */
  credentialInvalid: boolean;
  /** Whether the credential lookup failed, so connection state is unknown rather than absent. */
  credentialStatusUnavailable?: boolean;
  onViewDetails: () => void;
}

/** Every state that has a credential to talk about differs only by icon and copy. */
const StoredCredentialCard = ({
  iconName,
  editCredential,
  children,
}: {
  iconName: IconNames;
  editCredential: () => void;
  children: React.ReactNode;
}) => (
  <SectionCard
    iconName={iconName}
    cta={
      <Button onClick={editCredential} variant="subdued" icon="pencil">
        Edit
      </Button>
    }
  >
    {children}
  </SectionCard>
);

const MicrosoftGraphCard = ({
  credentialAdded,
  credentialInvalid,
  credentialStatusUnavailable = false,
  onViewDetails,
}: IMicrosoftGraphCardProps) => {
  // A failed lookup is not the same as no credential; saying "Connect" here would misreport a configured tenant.
  if (credentialStatusUnavailable) {
    return (
      <StoredCredentialCard iconName="warning" editCredential={onViewDetails}>
        Couldn&apos;t load the Microsoft Graph connection status.
      </StoredCredentialCard>
    );
  }

  if (!credentialAdded) {
    return (
      <SectionCard
        header="Microsoft Graph"
        cta={<Button onClick={onViewDetails}>Connect</Button>}
      >
        Add a Microsoft Entra app registration to sync Windows Autopilot devices
        to Fleet as pending hosts.
      </SectionCard>
    );
  }

  if (credentialInvalid) {
    return (
      <StoredCredentialCard iconName="error" editCredential={onViewDetails}>
        Microsoft Graph credential is invalid. Windows Autopilot devices
        won&apos;t sync to Fleet as pending hosts.
      </StoredCredentialCard>
    );
  }

  return (
    <StoredCredentialCard iconName="success" editCredential={onViewDetails}>
      Microsoft Graph connected.
    </StoredCredentialCard>
  );
};

export default MicrosoftGraphCard;
