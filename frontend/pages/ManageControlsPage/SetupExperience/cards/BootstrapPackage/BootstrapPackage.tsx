import { AxiosError, AxiosResponse } from "axios";
import React, { useMemo, useState } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import PageDescription from "components/PageDescription";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import {
  IEnrollSecret,
  IEnrollSecretsResponse,
} from "interfaces/enroll_secret";
import { IApiError } from "interfaces/errors";
import { ISoftwareTitle } from "interfaces/software";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import enrollSecretsAPI from "services/entities/enroll_secret";
import mdmAPI, {
  IGetBootstrapPackageMetadataResponse,
  IGetSetupExperienceScriptResponse,
  IGetSetupExperienceSoftwareResponse,
} from "services/entities/mdm";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import {
  getManualAgentInstallSetting,
  getBootstrapPackageManualEnrollmentSetting,
  getBootstrapPackageMigrationSetting,
} from "../../helpers";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";
import { getInstallSoftwareDuringSetupCount } from "../InstallSoftware/components/InstallSoftwareForm/helpers";

import BootstrapAdvancedOptions from "./components/BootstrapAdvancedOptions";
import PackageUploader from "./components/BootstrapPackageUploader";
import DeleteBootstrapPackageModal from "./components/DeleteBootstrapPackageModal";
import UploadedPackageView from "./components/UploadedPackageView";

const baseClass = "bootstrap-package";

// This is so large because we want to get all the software titles that are
// available for install so we can correctly display the selected count.
const PER_PAGE_SIZE = 3000;

const BootstrapPackage = ({
  currentTeamId,
  router,
}: ISetupExperienceCardProps) => {
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
    }
  );

  const { data: globalEnrollSecrets } = useQuery<
    IEnrollSecretsResponse,
    Error,
    IEnrollSecret[]
  >(["global secrets"], () => enrollSecretsAPI.getGlobalEnrollSecrets(), {
    enabled: currentTeamId === API_NO_TEAM_ID,
    select: (data: IEnrollSecretsResponse) => data.secrets,
  });

  const {
    data: teamConfig,
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: currentTeamId !== API_NO_TEAM_ID,
      select: (res) => res.fleet,
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
      // the manual enrollment and migration settings are kept, they do nothing
      // without a package. Manual agent install isn't, since hosts would get no fleetd.
      await mdmAPI.updateSetupExperienceSettings({
        fleet_id: currentTeamId,
        macos_manual_agent_install: false,
      });
      notify.success("Successfully deleted.");
    } catch (err) {
      notify.error("Couldn't delete. Please try again.", { response: err });
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

  // memoized since BootstrapAdvancedOptions resets its form whenever this changes
  const advancedOptionsInitialValues = useMemo(
    () => ({
      macos_manual_agent_install: getManualAgentInstallSetting(
        currentTeamId,
        globalConfig,
        teamConfig
      ),
      macos_enable_bootstrap_package_during_manual_enrollment: getBootstrapPackageManualEnrollmentSetting(
        currentTeamId,
        globalConfig,
        teamConfig
      ),
      macos_enable_bootstrap_package_during_migration: getBootstrapPackageMigrationSetting(
        currentTeamId,
        globalConfig,
        teamConfig
      ),
    }),
    [currentTeamId, globalConfig, teamConfig]
  );

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
          hasSetupExperienceSoftwareOrScript={
            hasSetupExperienceInstallSoftware || hasSetupExperienceScript
          }
          enrollSecret={
            (currentTeamId === API_NO_TEAM_ID
              ? globalEnrollSecrets?.[0]?.secret
              : teamConfig?.secrets?.[0]?.secret) ?? null
          }
          hasUploadedPackage={!noPackageUploaded}
          initialValues={advancedOptionsInitialValues}
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

    const mdmNotConfigured = !globalConfig?.mdm.enabled_and_configured;

    if (mdmNotConfigured) {
      return (
        <EmptyState
          variant="form"
          header="Additional configuration required"
          info={
            <>
              Turn on MDM to deploy a custom bootstrap package.
              <br />
              Supported on macOS.
            </>
          }
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
