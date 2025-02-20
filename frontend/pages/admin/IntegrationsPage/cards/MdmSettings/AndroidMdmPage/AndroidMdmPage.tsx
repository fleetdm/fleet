import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";

import TurnOffAndroidMdmModal from "./components/TurnOffAndroidMdmModal";

const baseClass = "android-mdm-page";

interface ITurnOnAndroidMdmProps {
  onClickConnect: () => void;
}

const TurnOnAndroidMdm = ({ onClickConnect }: ITurnOnAndroidMdmProps) => {
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
      <Button onClick={onClickConnect}>Connect</Button>
    </>
  );
};

interface ITurnOffAndroidMdmProps {
  onClickTurnOff: () => void;
}

const TurnOffAndroidMdm = ({ onClickTurnOff }: ITurnOffAndroidMdmProps) => {
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
        value="1234"
      />
      <Button onClick={onClickTurnOff}>Turn off Android MDM</Button>
    </>
  );
};

interface IAndroidMdmPageProps {
  router: InjectedRouter;
}

const AndroidMdmPage = ({ router }: IAndroidMdmPageProps) => {
  let { isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  isAndroidMdmEnabledAndConfigured = true;

  const [showTurnOffMdmModal, setShowTurnOffMdmModal] = useState(false);

  const onConnectMdm = () => {
    return false;
  };

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
          <TurnOnAndroidMdm onClickConnect={onConnectMdm} />
        ) : (
          <TurnOffAndroidMdm
            onClickTurnOff={() => setShowTurnOffMdmModal(true)}
          />
        )}
      </div>
      {showTurnOffMdmModal && (
        <TurnOffAndroidMdmModal onExit={() => setShowTurnOffMdmModal(false)} />
      )}
    </MainContent>
  );
};

export default AndroidMdmPage;
