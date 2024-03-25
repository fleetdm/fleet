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
import AppleBusinessManagerSection from "./components/AppleBusinessManagerSection/AppleBusinessManagerSection";
import IdpSection from "./components/IdpSection/IdpSection";

import EulaSection from "./components/EulaSection/EulaSection";

const baseClass = "automatic-enrollment";

interface IAutomaticEnrollment {
  router: InjectedRouter;
}

const AutomaticEnrollment = ({ router }: IAutomaticEnrollment) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const { isLoading: isLoadingMdmApple, error: errorMdmApple } = useQuery<
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

  if (isLoadingMdmApple) {
    return (
      <div className={baseClass}>
        <Spinner />
      </div>
    );
  }

  if (errorMdmApple?.status === 404) {
    return (
      <EmptyTable
        header="Automatic enrollment for macOS hosts"
        info="Connect Fleet to the Apple Push Certificates Portal to get started."
        primaryButton={<Button onClick={onClickConnect}>Connect</Button>}
        className={`${baseClass}__connect-message`}
      />
    );
  }

  if (errorMdmApple) {
    return <DataError />;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <AppleBusinessManagerSection router={router} />
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
