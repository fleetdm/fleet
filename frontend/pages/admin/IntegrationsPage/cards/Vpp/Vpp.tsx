import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import Card from "components/Card";
import SectionHeader from "components/SectionHeader";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { AppContext } from "context/app";

const baseClass = "vpp";

interface IVppCardProps {
  isAppleMdmOn: boolean;
  isVppOn: boolean;
  router: InjectedRouter;
}

const VppCard = ({ isAppleMdmOn, isVppOn, router }: IVppCardProps) => {
  const nagivateToMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  const navigateToVppSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_VPP_SETUP);
  };

  const appleMdmDiabledContent = (
    <div className={`${baseClass}__mdm-disabled-content`}>
      <div>
        <h3>Volume Purchasing Program (VPP)</h3>
        <p>
          To enable Volume Purchasing Program (VPP) for macOS devices, first
          turn on macOS MDM.
        </p>
      </div>
      <Button onClick={nagivateToMdm} variant="text-link">
        Turn on macOS MDM
      </Button>
    </div>
  );
  const isOnContent = (
    <div className={`${baseClass}__vpp-on-content`}>
      <p>
        <span>
          <Icon name="success" />
          Volume Purchasing Program (VPP) enabled.
        </span>
      </p>
      <Button onClick={navigateToVppSetup} variant="text-icon">
        <Icon name="pencil" />
        Edit
      </Button>
    </div>
  );

  const isOffContent = (
    <div className={`${baseClass}__vpp-off-content`}>
      <div>
        <h3>Volume Purchasing Program (VPP)</h3>
        <p>
          Install apps from Apple&apos;s App Store purchased through Apple
          Business Manager.
        </p>
      </div>
      <Button onClick={navigateToVppSetup} variant="brand">
        Enable
      </Button>
    </div>
  );

  const renderCardContent = () => {
    if (!isAppleMdmOn) {
      return appleMdmDiabledContent;
    }

    return isVppOn ? isOnContent : isOffContent;
  };

  return (
    <Card className={`${baseClass}__card`} color="gray">
      {renderCardContent()}
    </Card>
  );
};

interface IVppProps {
  router: InjectedRouter;
}

const Vpp = ({ router }: IVppProps) => {
  const { config } = useContext(AppContext);

  return (
    <div className={baseClass}>
      <SectionHeader title="Volume Purchasing Program (VPP)" />
      <VppCard
        isAppleMdmOn={!!config?.mdm.enabled_and_configured}
        isVppOn={false}
        router={router}
      />
    </div>
  );
};

export default Vpp;
