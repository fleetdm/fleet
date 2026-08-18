import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";
import microsoftGraphCredentialsAPI, {
  IGetMicrosoftGraphCredentialsResponse,
} from "services/entities/microsoft_graph_credentials";

import WindowsAutomaticEnrollmentCard from "./WindowsAutomaticEnrollmentCard";
import MicrosoftGraphCard from "./MicrosoftGraphCard";

const baseClass = "microsoft-entra-section";

interface IMicrosoftEntraSectionProps {
  router: InjectedRouter;
  windowsMdmEnabled: boolean;
  tenantAdded: boolean;
  isPremiumTier: boolean;
}

const MicrosoftEntraSection = ({
  router,
  windowsMdmEnabled,
  tenantAdded,
  isPremiumTier,
}: IMicrosoftEntraSectionProps) => {
  // The credential is not on the app config, so the card's state comes from its own endpoint. The endpoint is premium
  // only, so on Free the section renders the upsell instead and never asks.
  const { data: credentialsResponse } = useQuery<
    IGetMicrosoftGraphCredentialsResponse,
    Error
  >(
    ["microsoft-graph-credentials"],
    () => microsoftGraphCredentialsAPI.getCredentials(),
    {
      enabled: isPremiumTier,
      refetchOnWindowFocus: false,
    }
  );

  // Fleet stores at most one credential.
  const credential = credentialsResponse?.microsoft_graph_credentials?.[0];

  const navigateToWindowsEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  const navigateToMicrosoftGraph = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MICROSOFT_GRAPH);
  };

  return (
    <SettingsSection title="Microsoft Entra" className={baseClass}>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <div className={`${baseClass}__content`}>
          <WindowsAutomaticEnrollmentCard
            windowsMdmEnabled={windowsMdmEnabled}
            tenantAdded={tenantAdded}
            viewDetails={navigateToWindowsEnrollment}
          />
          <MicrosoftGraphCard
            credentialAdded={!!credential}
            credentialInvalid={!!credential?.credential_invalid}
            viewDetails={navigateToMicrosoftGraph}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default MicrosoftEntraSection;
