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
  const {
    data: credentialsResponse,
    isLoading: isLoadingCredentials,
    isError: isCredentialsError,
  } = useQuery<IGetMicrosoftGraphCredentialsResponse, Error>(
    ["microsoft-graph-credentials"],
    () => microsoftGraphCredentialsAPI.getCredentials(),
    {
      enabled: isPremiumTier,
      // Matches MicrosoftGraphPage, which shares this query key: the card's connected/invalid state should refresh when
      // the admin returns from the Entra portal.
      refetchOnWindowFocus: true,
    }
  );

  // Fleet stores at most one credential.
  const credential = credentialsResponse?.microsoft_graph_credentials[0];

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
        <>
          <WindowsAutomaticEnrollmentCard
            windowsMdmEnabled={windowsMdmEnabled}
            tenantAdded={tenantAdded}
            onViewDetails={navigateToWindowsEnrollment}
          />
          {!isLoadingCredentials && (
            <MicrosoftGraphCard
              credentialAdded={!!credential}
              credentialInvalid={!!credential?.credential_invalid}
              credentialStatusUnavailable={isCredentialsError}
              onViewDetails={navigateToMicrosoftGraph}
            />
          )}
        </>
      )}
    </SettingsSection>
  );
};

export default MicrosoftEntraSection;
