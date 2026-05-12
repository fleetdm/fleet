import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError, AxiosResponse } from "axios";

import PATHS from "router/paths";
import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { ISoftwareTitle } from "interfaces/software";
import mdmAPI, {
  IGetBootstrapPackageMetadataResponse,
  IGetSetupExperienceScriptResponse,
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { NotificationContext } from "context/notification";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import Spinner from "components/Spinner";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import CustomLink from "components/CustomLink";

import PackageUploader from "./components/BootstrapPackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeleteBootstrapPackageModal from "./components/DeleteBootstrapPackageModal";
import BootstrapAdvancedOptions from "./components/BootstrapAdvancedOptions";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { getInstallSoftwareDuringSetupCount } from "../InstallSoftware/components/InstallSoftwareForm/helpers";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";
import getManualAgentInstallSetting from "../../helpers";

const baseClass = "bootstrap-package";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

const BootstrapPackage = ({
  currentTeamId,
  router,
}: ISetupExperienceCardProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [
    selectedManualAgentInstall,
    setSelectedManualAgentInstall,
  ] = useState<boolean>(false);
  const [
    showDeleteBootstrapPackageModal,
    setShowDeleteBootstrapPackageModal,
  ] = useState(false);

  const { data: macSoftwareTitles, isLoading: isLoadingSoftware } = useQuery<
    IGetSetupExperienceSoftwareResponse,
    AxiosError,
    ISoftwareTitle[] | null
  >(
    ["install-software", currentTeamId],
    () =>
      mdmAPI.getSetupExperienceSoftware({
        platform: "macos",
        fleet_id: currentTeamId,
        per_page: PER_PAGE_SIZE,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.software_titles,
    }
  );

  const { data: script, isLoading: isLoadingScript } = useQuery<
    IGetSetupExperienceScriptResponse,
    AxiosError
  >(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const {
    data: globalConfig,
    isLoading: isLoadingGlobalConfig,
    refetch: refetchGlobalConfig,
  } = useQuery<IConfig, Error>(
    ["config", currentTeamId],
    () => configAPI.loadAll(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      onSuccess: (data) => {
        if (currentTeamId === API_NO_TEAM_ID) {
          setSelectedManualAgentInstall(
            getManualAgentInstallSetting(currentTeamId, data)
          );
        }
      },
    }
  );

  const {
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: currentTeamId !== API_NO_TEAM_ID,
      select: (res) => res.fleet,
      onSuccess: (data) => {
        setSelectedManualAgentInstall(
          getManualAgentInstallSetting(currentTeamId, undefined, data)
        );
      },
    }
  );

  const {
    data: bootstrapMetadata,
    isLoading: isloadingBootstrapMetadata,
    error: errorBootstrapMetadata,
    refetch: refretchBootstrapMetadata,
  } = useQuery<IGetBootstrapPackageMetadataResponse, AxiosResponse<IApiError>>(
    ["bootstrap-metadata", currentTeamId],
    () => mdmAPI.getBootstrapPackageMetadata(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      cacheTime: 0,
    }
  );

  const onUpload = () => {
    refretchBootstrapMetadata();
  };

  const onDelete = async () => {
    try {
      await mdmAPI.deleteBootstrapPackage(currentTeamId);
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        macos_manual_agent_install: false,
      });
      renderFlash("success", "Successfully deleted.");
    } catch {
      renderFlash("error", "Couldn't delete. Please try again.");
    } finally {
      setShowDeleteBootstrapPackageModal(false);
      refretchBootstrapMetadata();
      if (currentTeamId !== API_NO_TEAM_ID) {
        refetchTeamConfig();
      } else {
        refetchGlobalConfig();
      }
    }
  };

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noPackageUploaded =
    (errorBootstrapMetadata && errorBootstrapMetadata.status === 404) ||
    !bootstrapMetadata;
  const hasSetupExperienceInstallSoftware =
    getInstallSoftwareDuringSetupCount(macSoftwareTitles) !== 0;
  const hasSetupExperienceScript = !!script;

  const renderBootstrapView = () => {
    const bootstrapPackageView = noPackageUploaded ? (
      <PackageUploader currentTeamId={currentTeamId} onUpload={onUpload} />
    ) : (
      <UploadedPackageView
        bootstrapPackage={bootstrapMetadata}
        currentTeamId={currentTeamId}
        onDelete={() => setShowDeleteBootstrapPackageModal(true)}
      />
    );

    return (
      <>
        {bootstrapPackageView}
        <BootstrapAdvancedOptions
          currentTeamId={currentTeamId}
          disableInstallManually={
            noPackageUploaded ||
            hasSetupExperienceInstallSoftware ||
            hasSetupExperienceScript
          }
          selectManualAgentInstall={selectedManualAgentInstall}
          onChange={(manualAgentInstall) => {
            setSelectedManualAgentInstall(manualAgentInstall);
          }}
        />
      </>
    );
  };

  const isLoading =
    isloadingBootstrapMetadata ||
    isLoadingGlobalConfig ||
    isLoadingTeamConfig ||
    isLoadingScript ||
    isLoadingSoftware;

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    const mdmNotConfigured = !(
      globalConfig?.mdm.enabled_and_configured &&
      globalConfig?.mdm.apple_bm_enabled_and_configured
    );

    if (mdmNotConfigured) {
      return (
        <EmptyState
          variant="form"
          header="Additional configuration required"
          info="Supported on macOS. To customize, first turn on automatic enrollment."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      );
    }
    return renderBootstrapView();
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="Bootstrap package"
        details={
          <CustomLink
            newTab
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-experience/bootstrap-package`}
            text="Preview end user experience"
          />
        }
      />
      <PageDescription
        variant="right-panel"
        content="Upload a bootstrap package to install a configuration management tool (e.g. Munki, Chef, or Puppet) on macOS hosts that automatically enroll to Fleet."
      />
      <SetupExperienceContentContainer>
        {renderContent()}
      </SetupExperienceContentContainer>
      {showDeleteBootstrapPackageModal && (
        <DeleteBootstrapPackageModal
          onDelete={onDelete}
          onCancel={() => setShowDeleteBootstrapPackageModal(false)}
        />
      )}
    </section>
  );
};

export default BootstrapPackage;
