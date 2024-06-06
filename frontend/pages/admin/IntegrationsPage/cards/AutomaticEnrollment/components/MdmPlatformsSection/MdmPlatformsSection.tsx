import React, { useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";
import { IMdmAppleBm } from "interfaces/mdm";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { AppContext } from "context/app";

import DataError from "components/DataError";
import Spinner from "components/Spinner/Spinner";
import SectionHeader from "components/SectionHeader";

import WindowsAutomaticEnrollmentCard from "./components/WindowsAutomaticEnrollmentCard";
import AppleAutomaticEnrollmentCard from "./components/AppleAutomaticEnrollmentCard";

const baseClass = "mdm-platforms-section";

interface IMdmPlatformsSectionProps {
  router: InjectedRouter;
}

const MdmPlatformsSection = ({ router }: IMdmPlatformsSectionProps) => {
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
    }
  );

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

  const showMdmAppleBmError =
    errorMdmAppleBm &&
    // API returns a 404 error if ABM is not configured yet
    errorMdmAppleBm.status !== 404 &&
    // API returns a 400 error if ABM credentials are invalid
    errorMdmAppleBm.status !== 400; // TODO: does this still signal expire/invalid credentials? do we need any special error handling? can anything else result in 400?

  if (showMdmAppleBmError) {
    return <DataError />;
  }

  return (
    <div className={baseClass}>
      <SectionHeader title="Apple Business Manager" />
      <AppleAutomaticEnrollmentCard
        viewDetails={navigateToAppleAutomaticEnrollment}
        turnOn={
          !config?.mdm.enabled_and_configured
            ? navigateToApplePushCertSetup
            : undefined
        }
        configured={!!config?.mdm.apple_bm_enabled_and_configured}
      />
      <WindowsAutomaticEnrollmentCard
        viewDetails={navigateToWindowsAutomaticEnrollment}
      />
    </div>
  );
};

export default MdmPlatformsSection;
