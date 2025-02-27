import React, { useContext, useEffect, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import endpoints from "utilities/endpoints";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import mdmAndroidAPI from "services/entities/mdm_android";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import TurnOffAndroidMdmModal from "./components/TurnOffAndroidMdmModal";

const baseClass = "android-mdm-page";

const POPUP_WIDTH = 885;
const POPUP_HEIGHT = 600;

interface ITurnOnAndroidMdmProps {
  router: InjectedRouter;
}

const TurnOnAndroidMdm = ({ router }: ITurnOnAndroidMdmProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [fetchingSignupUrl, setFetchingSignupUrl] = useState(false);
  const [setupSse, setSetupSse] = useState(false);

  useEffect(() => {
    if (setupSse) {
      const eventSource = new EventSource(endpoints.MDM_ANDOIR_SSE_URL);

      eventSource.onmessage = (event) => {
        console.log("onmessage", event);
        const data = JSON.parse(event.data);
        console.log(data);
        if (data === "Android Enterprise successfully connected") {
          renderFlash("success", "Android MDM turned on successfully.", {
            persistOnPageChange: true,
          });
          setSetupSse(false);
          eventSource.close();
          router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
        }
      };

      eventSource.onerror = () => {
        console.log("error");
        renderFlash("error", "Couldn't turn on Android MDM. Please try again.");
        setSetupSse(false);
        eventSource.close();
      };

      return () => {
        eventSource.close();
      };
    }
  }, [setupSse, router, renderFlash]);

  const onConnectMdm = async () => {
    setFetchingSignupUrl(true);
    try {
      const res = await mdmAndroidAPI.getSignupUrl();

      // Calculate the center position
      const left = window.screenX + (window.innerWidth - POPUP_WIDTH) / 2;
      const top = window.screenY + (window.innerHeight - POPUP_HEIGHT) / 2;

      // TODO: set up SSE for successful android mdm turned on here.
      window.open(
        res.android_enterprise_signup_url,
        "_blank",
        `width=${POPUP_WIDTH},height=${POPUP_HEIGHT},top=${top},left=${left}`
      );
      setSetupSse(true);
    } catch (e) {
      renderFlash("error", "Couldn't connect. Please try again");
    }
    setFetchingSignupUrl(false);
  };

  return (
    <>
      <div className={`${baseClass}__turn-on-description`}>
        <p>Connect Android Enterprise to turn on Android MDM. </p>
        <CustomLink
          text="Learn More"
          newTab
          url="https://fleetdm.com/learn-more-about/how-to-connect-android-enterprise"
        />
      </div>
      <Button isLoading={fetchingSignupUrl} onClick={onConnectMdm}>
        Connect
      </Button>
    </>
  );
};

interface ITurnOffAndroidMdmProps {
  onClickTurnOff: () => void;
}

const TurnOffAndroidMdm = ({ onClickTurnOff }: ITurnOffAndroidMdmProps) => {
  const { data, isLoading, isError } = useQuery(
    ["androidEnterprise"],
    () => mdmAndroidAPI.getAndroidEnterprise(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return <DataError />;
  }

  if (!data) return null;

  return (
    <>
      <DataSet
        title={
          <TooltipWrapper
            position="top"
            tipContent={
              <>
                Android Enterprise in{" "}
                <CustomLink
                  newTab
                  text="Google Admin Console"
                  url="https://fleetdm.com/learn-more-about/google-admin-emm"
                  variant="tooltip-link"
                />
              </>
            }
          >
            Android Enterprise Id
          </TooltipWrapper>
        }
        value={data.android_enterprise_id}
      />
      <Button onClick={onClickTurnOff}>Turn off Android MDM</Button>
    </>
  );
};

interface IAndroidMdmPageProps {
  router: InjectedRouter;
}

const AndroidMdmPage = ({ router }: IAndroidMdmPageProps) => {
  const { isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const [showTurnOffMdmModal, setShowTurnOffMdmModal] = useState(false);

  return (
    <MainContent className={baseClass}>
      <BackLink
        text="Back to MDM"
        path={PATHS.ADMIN_INTEGRATIONS_MDM}
        className={`${baseClass}__back-to-mdm`}
      />
      <h1>Android Enterprise</h1>

      <div className={`${baseClass}__content`}>
        {!isAndroidMdmEnabledAndConfigured ? (
          <TurnOnAndroidMdm router={router} />
        ) : (
          <TurnOffAndroidMdm
            onClickTurnOff={() => setShowTurnOffMdmModal(true)}
          />
        )}
      </div>
      {showTurnOffMdmModal && (
        <TurnOffAndroidMdmModal
          router={router}
          onExit={() => setShowTurnOffMdmModal(false)}
        />
      )}
    </MainContent>
  );
};

export default AndroidMdmPage;
