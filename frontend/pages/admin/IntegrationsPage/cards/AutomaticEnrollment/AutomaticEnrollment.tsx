import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { IMdmApple } from "interfaces/mdm";
import mdmAppleAPI from "services/entities/mdm_apple";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PremiumFeatureMessage from "components/PremiumFeatureMessage/PremiumFeatureMessage";
import EmptyTable from "components/EmptyTable/EmptyTable";
import Button from "components/buttons/Button/Button";

import MdmPlatformsSection from "./components/MdmPlatformsSection/MdmPlatformsSection";
import DefaultTeamSection from "./components/DefaultTeamSection/DefaultTeamSection";
import IdpSection from "./components/IdpSection/IdpSection";
import EulaSection from "./components/EulaSection/EulaSection";

const baseClass = "automatic-enrollment";

interface IAutomaticEnrollment {
  router: InjectedRouter;
}

const AutomaticEnrollment = ({ router }: IAutomaticEnrollment) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const { isLoading: isLoadingAPNInfo, error: errorAPNInfo } = useQuery<
    IMdmApple,
    AxiosError
  >(["appleAPNInfo"], () => mdmAppleAPI.getAppleAPNInfo(), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: config?.mdm.enabled_and_configured,
  });

  const onClickConnect = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  if (!isPremiumTier) return <PremiumFeatureMessage />;

  if (isLoadingAPNInfo) {
    return (
      <div className={baseClass}>
        <Spinner />
      </div>
    );
  }

  if (errorAPNInfo?.status === 404) {
    return (
      <EmptyTable
        header="Automatic enrollment for macOS hosts"
        info="Connect Fleet to the Apple Push Certificates Portal to get started."
        primaryButton={<Button onClick={onClickConnect}>Connect</Button>}
        className={`${baseClass}__connect-message`}
      />
    );
  }

  if (errorAPNInfo) {
    return <DataError />;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <MdmPlatformsSection router={router} />
      </div>
      {!!config?.mdm.apple_bm_enabled_and_configured && (
        <div className={`${baseClass}__section`}>
          <DefaultTeamSection />
        </div>
      )}
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
