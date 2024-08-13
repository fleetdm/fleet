import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import mdmAppleAPI, { IGetVppInfoResponse } from "services/entities/mdm_apple";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SettingsSection from "pages/admin/components/SettingsSection";

const baseClass = "vpp-section";

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
          To enable Volume Purchasing Program (VPP), first turn on Apple (macOS,
          iOS, iPadOS) MDM.
        </p>
      </div>
      <Button onClick={nagivateToMdm} variant="text-link">
        Turn on MDM
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

interface IVppSectionProps {
  router: InjectedRouter;
}

const VppSection = ({ router }: IVppSectionProps) => {
  const { config } = useContext(AppContext);

  const { data: vppData, error: vppError, isLoading, isError } = useQuery<
    IGetVppInfoResponse,
    AxiosError
  >("vppInfo", () => mdmAppleAPI.getVppInfo(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    retry: false,
  });

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError && vppError?.status !== 404) {
      return <DataError />;
    }

    return (
      <VppCard
        isAppleMdmOn={!!config?.mdm.enabled_and_configured}
        isVppOn={!!vppData && vppError?.status !== 404}
        router={router}
      />
    );
  };

  return (
    <SettingsSection
      title="Volume Purchasing Program (VPP)"
      className={baseClass}
    >
      <>{renderContent()}</>
    </SettingsSection>
  );
};

export default VppSection;
