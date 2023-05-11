import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { AppContext } from "context/app";
import { IMdmApple } from "interfaces/mdm";
import mdmAppleAPI from "services/entities/mdm_apple";

import PremiumFeatureMessage from "components/PremiumFeatureMessage/PremiumFeatureMessage";
import EmptyTable from "components/EmptyTable/EmptyTable";
import Button from "components/buttons/Button/Button";
import AppleBusinessManagerSection from "./components/AppleBusinessManagerSection/AppleBusinessManagerSection";
import IdpSection from "./components/IdpSection/IdpSection";
import EulaSection from "./components/EulaSection/EulaSection";

const baseClass = "automatic-enrollment";

const AutomaticEnrollment = () => {
  const { config, isPremiumTier } = useContext(AppContext);

  const {
    data: appleAPNInfo,
    isLoading: isLoadingMdmApple,
    error: errorMdmApple,
  } = useQuery<IMdmApple, AxiosError>(
    ["appleAPNInfo"],
    () => mdmAppleAPI.getAppleAPNInfo(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
      enabled: config?.mdm.enabled_and_configured,
    }
  );

  if (!isPremiumTier) return <PremiumFeatureMessage />;

  // TODO: figure out correct condition
  if (!appleAPNInfo) {
    return (
      <EmptyTable
        header="Automatic enrollment for macOS hosts"
        info="Connect Fleet to the Apple Push Certificates Portal to get started."
        primaryButton={<Button>Connect</Button>}
        className={`${baseClass}__connect-message`}
      />
    );
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <AppleBusinessManagerSection />
      </div>
      <div className={`${baseClass}__section`}>
        <IdpSection />
      </div>
      <div className={`${baseClass}__section`}>
        <EulaSection />
      </div>
    </div>
  );
};

export default AutomaticEnrollment;
