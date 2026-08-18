import React from "react";

import Button from "components/buttons/Button";
import SectionCard from "../../SectionCard";

const baseClass = "microsoft-graph-card";

interface IMicrosoftGraphCardProps {
  /** Whether a Graph credential is stored. */
  credentialAdded: boolean;
  /** Whether the stored credential was rejected by Entra on the last sync. */
  credentialInvalid: boolean;
  /** Whether the credential lookup failed, so connection state is unknown rather than absent. */
  credentialStatusUnavailable?: boolean;
  viewDetails: () => void;
}

const MicrosoftGraphNotAddedCard = ({
  addCredential,
}: {
  addCredential: () => void;
}) => (
  <SectionCard
    className={baseClass}
    header="Microsoft Graph"
    cta={<Button onClick={addCredential}>Connect</Button>}
  >
    Add a Microsoft Entra app registration to sync Windows Autopilot devices to
    Fleet as pending hosts.
  </SectionCard>
);

const MicrosoftGraphUnavailableCard = ({
  viewDetails,
}: {
  viewDetails: () => void;
}) => (
  <SectionCard
    className={baseClass}
    iconName="warning"
    cta={
      <Button onClick={viewDetails} variant="subdued" icon="pencil">
        Edit
      </Button>
    }
  >
    Couldn&apos;t load the Microsoft Graph connection status.
  </SectionCard>
);

const MicrosoftGraphAddedCard = ({
  editCredential,
}: {
  editCredential: () => void;
}) => (
  <SectionCard
    className={baseClass}
    iconName="success"
    cta={
      <Button onClick={editCredential} variant="subdued" icon="pencil">
        Edit
      </Button>
    }
  >
    Microsoft Graph connected.
  </SectionCard>
);

const MicrosoftGraphInvalidCard = ({
  editCredential,
}: {
  editCredential: () => void;
}) => (
  <SectionCard
    className={baseClass}
    iconName="error"
    cta={
      <Button onClick={editCredential} variant="subdued" icon="pencil">
        Edit
      </Button>
    }
  >
    Microsoft Graph credential is invalid. Windows Autopilot devices won&apos;t
    sync to Fleet as pending hosts.
  </SectionCard>
);

const MicrosoftGraphCard = ({
  credentialAdded,
  credentialInvalid,
  credentialStatusUnavailable = false,
  viewDetails,
}: IMicrosoftGraphCardProps) => {
  // A failed lookup is not the same as no credential; saying "Connect" here would misreport a configured tenant.
  if (credentialStatusUnavailable) {
    return <MicrosoftGraphUnavailableCard viewDetails={viewDetails} />;
  }

  if (!credentialAdded) {
    return <MicrosoftGraphNotAddedCard addCredential={viewDetails} />;
  }

  if (credentialInvalid) {
    return <MicrosoftGraphInvalidCard editCredential={viewDetails} />;
  }

  return <MicrosoftGraphAddedCard editCredential={viewDetails} />;
};

export default MicrosoftGraphCard;
