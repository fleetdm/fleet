import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";
import { IMdmAppleBm } from "interfaces/mdm";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import DataError from "components/DataError";
import Spinner from "components/Spinner/Spinner";
import SectionHeader from "components/SectionHeader";

import WindowsAutomaticEnrollmentCard from "./components/WindowsAutomaticEnrollmentCard/WindowsAutomaticEnrollmentCard";
import AppleAutomaticEnrollmentCard from "./components/AppleAutomaticEnrollmentCard";

const baseClass = "apple-business-manager-section";

interface IABMKeys {
  decodedPublic: string;
  decodedPrivate: string;
}

interface IAppleBusinessManagerSectionProps {
  router: InjectedRouter;
}

const AppleBusinessManagerSection = ({
  router,
}: IAppleBusinessManagerSectionProps) => {
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [defaultTeamName, setDefaultTeamName] = useState("No team");
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const {
    data: mdmAppleBm,
    isLoading: isLoadingMdmAppleBm,
    error: errorMdmAppleBm,
  } = useQuery<IMdmAppleBm, AxiosError, IMdmAppleBm>(
    ["mdmAppleBmAPI"],
    () => mdmAppleBmAPI.getAppleBMInfo(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
      onSuccess: (appleBmData) => {
        setDefaultTeamName(appleBmData.default_team ?? "No team");
      },
    }
  );

  const {
    data: keys,
    error: fetchKeysError,
    isFetching: isFetchingKeys,
  } = useQuery<IABMKeys, Error>(["keys"], () => mdmAppleBmAPI.loadKeys(), {
    refetchOnWindowFocus: false,
    retry: false,
  });

  const toggleEditTeamModal = () => {
    setShowEditTeamModal(!showEditTeamModal);
  };

  const navigateToWindowsAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  const navigateToAppleAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_APPLE);
  };

  const navigateToApplePushCertSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  if (isLoadingMdmAppleBm) {
    return (
      <div className={baseClass}>
        <Spinner />
      </div>
    );
  }

  if (errorMdmAppleBm) {
    // TODO: other error handling (expired case from above)
    return <DataError />;
  }

  console.log("config", config);

  return (
    <div className={baseClass}>
      <SectionHeader title="Apple Business Manager" />
      {/* {isLoadingMdmAppleBm ? <Spiner /> : renderAppleBMInfo()} */}
      <AppleAutomaticEnrollmentCard
        viewDetails={navigateToAppleAutomaticEnrollment}
        turnOn={!mdmAppleBm ? navigateToApplePushCertSetup : undefined}
        configured={!!config?.mdm.apple_bm_enabled_and_configured}
      />
      <WindowsAutomaticEnrollmentCard
        viewDetails={navigateToWindowsAutomaticEnrollment}
      />
    </div>
  );
};

export default AppleBusinessManagerSection;
