import React from "react";
import { useQuery } from "react-query";

import BackButton from "components/BackButton";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import PATHS from "router/paths";
import mdmAndroidAPI, {
  IGetZeroTouchConfigurationResponse,
} from "services/entities/mdm_android";

const baseClass = "android-zero-touch-page";

const AndroidZeroTouchPage = () => {
  const { data: zeroTouchConfig, isLoading, isError } = useQuery<
    IGetZeroTouchConfigurationResponse,
    Error
  >(
    ["android-zero-touch-configuration"],
    () => mdmAndroidAPI.getZeroTouchConfiguration(),
    { refetchOnWindowFocus: false }
  );

  const renderCodeBlock = () => {
    if (isLoading) {
      return (
        <div
          className={`${baseClass}__dpc-extras-code ${baseClass}__dpc-extras-code--loading`}
        >
          <Spinner />
        </div>
      );
    }

    if (isError || !zeroTouchConfig) {
      return <DataError />;
    }

    return (
      <pre className={`${baseClass}__dpc-extras-code`}>
        <code>{zeroTouchConfig.dpc_extras}</code>
      </pre>
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__header-links`}>
        <BackButton
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
      </div>
      <h1>Android zero-touch</h1>
      <div className={`${baseClass}__description`}>
        To connect Fleet to Android zero-touch, go to the{" "}
        <CustomLink
          url="https://fleetdm.com/learn-more-about/android-zero-touch-portal"
          text="Android zero-touch portal"
          newTab
        />
      </div>
      <p className={`${baseClass}__enrollment-info`}>
        Android hosts will automatically enroll to the <b>Unassigned</b> fleet.
        Changing fleets is coming soon.
      </p>
      <div className={`${baseClass}__dpc-extras`}>
        <div className={`${baseClass}__dpc-extras-header`}>
          <span className={`${baseClass}__dpc-extras-label`}>DPC extras</span>
          {zeroTouchConfig && (
            <CopyButton
              copyText={zeroTouchConfig.dpc_extras}
              variant="secondary"
            />
          )}
        </div>
        {renderCodeBlock()}
      </div>
      {!isLoading && !isError && zeroTouchConfig && (
        <p className={`${baseClass}__instructions`}>
          Select <b>Add configuration</b>, pick <b>Android Device Policy</b> as
          your <b>EMM DPC</b>, and paste this JSON into <b>DPC extras</b>.
        </p>
      )}
    </MainContent>
  );
};

export default AndroidZeroTouchPage;
