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
        // Cards render as direct children of SettingsSection so its vertical-card-layout provides the standard
        // spacing, the same way MdmSettingsSection lays out the Apple/Windows/Android cards.
        <>
          <WindowsAutomaticEnrollmentCard
            windowsMdmEnabled={windowsMdmEnabled}
            tenantAdded={tenantAdded}
            viewDetails={navigateToWindowsEnrollment}
          />
          {/* Until the query settles, the card cannot tell "no credential" from "not loaded yet", and rendering the
              Connect state would misreport a configured tenant as disconnected. */}
          {!isLoadingCredentials && (
            <MicrosoftGraphCard
              credentialAdded={!!credential}
              credentialInvalid={!!credential?.credential_invalid}
              credentialStatusUnavailable={isCredentialsError}
              viewDetails={navigateToMicrosoftGraph}
            />
          )}
        </>
      )}
    </SettingsSection>
  );
};

export default MicrosoftEntraSection;
