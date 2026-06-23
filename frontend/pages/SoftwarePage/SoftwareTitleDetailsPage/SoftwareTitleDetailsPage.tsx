/** software/titles/:id */

import React, { useCallback, useContext, useState } from "react";
import { useQuery, useQueryClient } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import paths from "router/paths";
import useTeamIdParam from "hooks/useTeamIdParam";
import useGitOpsMode from "hooks/useGitOpsMode";
import { useSoftwareInstaller } from "hooks/useSoftwareInstallerMeta";
import { AppContext } from "context/app";
import { ignoreAxiosError } from "interfaces/errors";
import { ILabelSoftwareTitle } from "interfaces/label";
import { ISoftwareTitleDetails } from "interfaces/software";
import {
  APP_CONTEXT_ALL_TEAMS_ID,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import { canWriteSoftware } from "utilities/permissions/permissions";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import DetailsNoHosts from "../components/cards/DetailsNoHosts";
import SoftwareSummaryCard from "./SoftwareSummaryCard";
import SoftwareInstallerCard from "./SoftwareInstallerCard";
import LibraryItemAccordion, {
  LibraryItemLabelKind,
} from "./LibraryItemAccordion/LibraryItemAccordion";
import LibraryItemAccordionList from "./LibraryItemAccordion/LibraryItemAccordionList";
import EditSoftwareModal from "./EditSoftwareModal";
import { getDisplayedSoftwareName } from "../helpers";

const baseClass = "software-title-details-page";

interface ISoftwareTitleDetailsRouteParams {
  id: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareTitleDetailsRouteParams
>;

const SoftwareTitleDetailsPage = ({
  router,
  routeParams,
  location,
}: ISoftwareTitleDetailsPageProps) => {
  const {
    isPremiumTier,
    isOnGlobalTeam,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
    isTeamTechnician,
    currentUser,
    config,
  } = useContext(AppContext);
  const handlePageError = useErrorHandler();
  const queryClient = useQueryClient();

  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);
  const { gitOpsModeEnabled } = useGitOpsMode("software");
  const autoOpenGitOpsYamlModal =
    location.query.gitops_yaml === "true" && gitOpsModeEnabled;

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: true,
  });

  const canEditSoftware = canWriteSoftware(currentUser, currentTeamId ?? null);

  // gitOpsYamlParam URL Param controls whether the View Yaml modal is opened on page load
  // as it automatically opens from adding flow of custom software in gitOps mode
  const [showViewYamlModal, setShowViewYamlModal] = useState(
    autoOpenGitOpsYamlModal || false
  );

  // TODO #47622 preview — page-level state for opening the EditSoftwareModal
  // from the LibraryItemAccordion. Remove with the preview block.
  const [showLibraryEditModal, setShowLibraryEditModal] = useState(false);

  const {
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
    refetch: refetchSoftwareTitle,
  } = useQuery<
    ISoftwareTitleResponse,
    AxiosError,
    ISoftwareTitleDetails,
    IGetSoftwareTitleQueryKey[]
  >(
    [{ scope: "softwareById", softwareId, teamId: teamIdForApi }],
    ({ queryKey }) => softwareAPI.getSoftwareTitle(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      select: (data) => data.software_title,
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const isAvailableForInstall =
    !!softwareTitle?.software_package || !!softwareTitle?.app_store_app;

  // TODO #47622 preview — installer meta used to wire the EditSoftwareModal
  // from the accordion's label-count click. Remove with the preview block.
  const installerResult = useSoftwareInstaller(
    softwareTitle ?? ({} as ISoftwareTitleDetails)
  );

  const onToggleViewYaml = () => {
    setShowViewYamlModal(!showViewYamlModal);
  };

  const onDeleteInstaller = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: [{ scope: "software-titles" }] });
    queryClient.invalidateQueries({
      queryKey: [{ scope: "software-library" }],
    });

    if (softwareTitle?.versions?.length) {
      refetchSoftwareTitle();
      return;
    }

    // redirect to software library page if no versions are available
    router.push(
      getPathWithQueryParams(paths.SOFTWARE_LIBRARY, {
        fleet_id: teamIdForApi,
      })
    );
  }, [queryClient, refetchSoftwareTitle, router, softwareTitle, teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderSoftwareInstallerCard = (title: ISoftwareTitleDetails) => {
    const hasPermission = Boolean(
      isOnGlobalTeam ||
        isTeamAdmin ||
        isTeamMaintainer ||
        isTeamObserver ||
        isTeamTechnician
    );

    const showInstallerCard =
      currentTeamId !== APP_CONTEXT_ALL_TEAMS_ID &&
      hasPermission &&
      isAvailableForInstall;

    if (!showInstallerCard) {
      return null;
    }

    return (
      <SoftwareInstallerCard
        softwareTitle={title}
        softwareId={softwareId}
        teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
        onDelete={onDeleteInstaller}
        isLoading={isSoftwareTitleLoading}
        onToggleViewYaml={onToggleViewYaml}
        showViewYamlModal={showViewYamlModal}
      />
    );
  };

  const renderSoftwareSummaryCard = (title: ISoftwareTitleDetails) => {
    return (
      <SoftwareSummaryCard
        softwareTitle={title}
        softwareId={softwareId}
        teamId={teamIdForApi}
        isAvailableForInstall={isAvailableForInstall}
        isLoading={isSoftwareTitleLoading}
        router={router}
        refetchSoftwareTitle={refetchSoftwareTitle}
        onToggleViewYaml={onToggleViewYaml}
      />
    );
  };

  // TODO #47622 preview — remove before merging into main.
  // Renders a single LibraryItemAccordion from the active software_package or
  // app_store_app so design can review with real data; multi-row rendering
  // lands in #47623.
  const renderLibraryItemAccordionPreview = (title: ISoftwareTitleDetails) => {
    const pkg = title.software_package;
    const appStore = title.app_store_app;

    const statusPath = (software_status: "installed" | "pending" | "failed") =>
      getPathWithQueryParams(paths.MANAGE_HOSTS, {
        software_title_id: softwareId,
        software_status,
        fleet_id: currentTeamId ?? APP_CONTEXT_NO_TEAM_ID,
      });

    const installerMeta = installerResult?.meta;
    const isFma = installerMeta?.isFleetMaintainedApp ?? false;
    const isLatestFmaVersion = installerMeta?.isLatestFmaVersion ?? false;
    const isScriptPackage = installerResult?.cardInfo.isScriptPackage ?? false;

    interface ILabeledSource {
      labels_include_any: ILabelSoftwareTitle[] | null;
      labels_include_all: ILabelSoftwareTitle[] | null;
      labels_exclude_any: ILabelSoftwareTitle[] | null;
    }
    interface IPickedLabels {
      labels: ILabelSoftwareTitle[] | null;
      kind: LibraryItemLabelKind;
    }
    const pickLabels = (source: ILabeledSource): IPickedLabels => {
      if (source.labels_include_all?.length) {
        return { labels: source.labels_include_all, kind: "includeAll" };
      }
      if (source.labels_exclude_any?.length) {
        return { labels: source.labels_exclude_any, kind: "excludeAny" };
      }
      return { labels: source.labels_include_any, kind: "includeAny" };
    };

    if (appStore) {
      const { labels, kind } = pickLabels(appStore);
      const isAndroidPlayStoreApp = appStore.platform === "android";
      return (
        <LibraryItemAccordionList>
          <LibraryItemAccordion
            filename={appStore.name}
            version={appStore.latest_version}
            addedAt={appStore.created_at}
            installerType="app-store"
            androidPlayStoreId={
              isAndroidPlayStoreApp ? appStore.app_store_id : undefined
            }
            isScriptPackage={isScriptPackage}
            isActive
            badgeState="latest"
            labels={labels}
            labelKind={kind}
            canEditSoftware={canEditSoftware}
            installed={appStore.status?.installed ?? 0}
            pending={appStore.status?.pending ?? 0}
            failed={appStore.status?.failed ?? 0}
            installedPath={statusPath("installed")}
            pendingPath={statusPath("pending")}
            failedPath={statusPath("failed")}
            onLabelCountClick={() => setShowLibraryEditModal(true)}
            onLabelsClick={() => setShowLibraryEditModal(true)}
          />
        </LibraryItemAccordionList>
      );
    }

    if (!pkg) return null;
    const { labels, kind } = pickLabels(pkg);
    return (
      <LibraryItemAccordionList>
        <LibraryItemAccordion
          filename={pkg.name}
          version={pkg.version}
          addedAt={pkg.uploaded_at}
          isFma={isFma}
          isLatestFmaVersion={isLatestFmaVersion}
          isScriptPackage={isScriptPackage}
          isActive
          badgeState="latest"
          labels={labels}
          labelKind={kind}
          canEditSoftware={canEditSoftware}
          installed={pkg.status?.installed ?? 0}
          pending={
            (pkg.status?.pending_install ?? 0) +
            (pkg.status?.pending_uninstall ?? 0)
          }
          failed={
            (pkg.status?.failed_install ?? 0) +
            (pkg.status?.failed_uninstall ?? 0)
          }
          installedPath={statusPath("installed")}
          pendingPath={statusPath("pending")}
          failedPath={statusPath("failed")}
          hashSha256={pkg.hash_sha256 ?? null}
          downloadUrl={pkg.url}
          onLabelCountClick={() => setShowLibraryEditModal(true)}
          onLabelsClick={() => setShowLibraryEditModal(true)}
        />
        <LibraryItemAccordion
          filename="example-package-v2-really-long-package-name-to-see-what-happens-responsive-design.pkg"
          version="2.0.0"
          addedAt="2024-01-01T12:00:00Z"
          isFma={isFma}
          isLatestFmaVersion={isLatestFmaVersion}
          isScriptPackage={isScriptPackage}
          isActive={false}
          labels={labels}
          labelKind={kind}
          canEditSoftware={canEditSoftware}
          installed={pkg.status?.installed ?? 0}
          pending={
            (pkg.status?.pending_install ?? 0) +
            (pkg.status?.pending_uninstall ?? 0)
          }
          failed={
            (pkg.status?.failed_install ?? 0) +
            (pkg.status?.failed_uninstall ?? 0)
          }
          installedPath={statusPath("installed")}
          pendingPath={statusPath("pending")}
          failedPath={statusPath("failed")}
          hashSha256={pkg.hash_sha256 ?? null}
          downloadUrl={pkg.url}
          onLabelCountClick={() => setShowLibraryEditModal(true)}
          onLabelsClick={() => setShowLibraryEditModal(true)}
        />
      </LibraryItemAccordionList>
    );
  };

  const renderLibraryEditModal = (title: ISoftwareTitleDetails) => {
    if (!showLibraryEditModal || !installerResult) return null;
    const { meta } = installerResult;
    return (
      <EditSoftwareModal
        softwareId={softwareId}
        teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
        softwareInstaller={meta.softwareInstaller}
        refetchSoftwareTitle={refetchSoftwareTitle}
        onExit={() => setShowLibraryEditModal(false)}
        installerType={meta.installerType}
        openViewYamlModal={onToggleViewYaml}
        isFleetMaintainedApp={meta.isFleetMaintainedApp}
        isIosOrIpadosApp={meta.isIosOrIpadosApp}
        name={title.name}
        displayName={getDisplayedSoftwareName(title.name, title.display_name)}
        source={title.source}
        iconUrl={title.icon_url}
      />
    );
  };

  const renderContent = () => {
    if (isSoftwareTitleLoading) {
      return <Spinner />;
    }

    if (isSoftwareTitleError) {
      return (
        <DetailsNoHosts
          header="Software not detected"
          details="Expecting to see software? Check back later."
        />
      );
    }

    if (softwareTitle) {
      return (
        <>
          {renderSoftwareSummaryCard(softwareTitle)}
          {renderLibraryItemAccordionPreview(softwareTitle)}
          {renderSoftwareInstallerCard(softwareTitle)}
          {renderLibraryEditModal(softwareTitle)}
        </>
      );
    }

    return null;
  };

  return (
    <MainContent className={baseClass}>
      {isPremiumTier && !config?.partnerships?.enable_primo && (
        <TeamsHeader
          isOnGlobalTeam={isOnGlobalTeam}
          currentTeamId={currentTeamId}
          userTeams={userTeams}
          onTeamChange={onTeamChange}
        />
      )}
      <>{renderContent()}</>
    </MainContent>
  );
};

export default SoftwareTitleDetailsPage;
