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
import {
  aggregateInstallStatusCounts,
  IAppStoreApp,
  isIpadOrIphoneSoftwareSource,
  ISoftwarePackage,
  ISoftwareTitleDetails,
  NO_VERSION_OR_HOST_DATA_SOURCES,
} from "interfaces/software";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import { canWriteSoftware } from "utilities/permissions/permissions";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { notify } from "components/ToastNotification";
import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import TeamsHeader from "components/TeamsHeader";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import DetailsNoHosts from "../components/cards/DetailsNoHosts";
import SoftwareSummaryCard from "./SoftwareSummaryCard";
import LibraryItemAccordion, {
  LibraryItemLabelKind,
} from "./LibraryItemAccordion/LibraryItemAccordion";
import LibraryItemAccordionList from "./LibraryItemAccordion/LibraryItemAccordionList";
import EditSoftwareModal from "./EditSoftwareModal";
import DeleteSoftwareModal from "./DeleteSoftwareModal";
import VersionsModal from "./VersionsModal";
import { getDisplayedSoftwareName } from "../helpers";
import { buildLibraryVersionRows } from "./helpers";
import TitleVersionsTable from "./TitleVersionsTable";
import ViewYamlModal from "./ViewYamlModal";

const baseClass = "software-title-details-page";

const pickLabels = (
  source: ISoftwarePackage | IAppStoreApp
): { labels: ILabelSoftwareTitle[] | null; kind: LibraryItemLabelKind } => {
  if (source.labels_include_all?.length) {
    return { labels: source.labels_include_all, kind: "includeAll" };
  }
  if (source.labels_exclude_any?.length) {
    return { labels: source.labels_exclude_any, kind: "excludeAny" };
  }
  return { labels: source.labels_include_any, kind: "includeAny" };
};

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
  const { isPremiumTier, isOnGlobalTeam, currentUser, config } = useContext(
    AppContext
  );
  const handlePageError = useErrorHandler();
  const queryClient = useQueryClient();

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

  const [showLibraryEditModal, setShowLibraryEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  // Page-owned so both the Actions menu and the Library accordion badge open
  // the same Versions modal.
  const [showVersionsModal, setShowVersionsModal] = useState(false);

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

  // Mints a one-shot download token and triggers the browser download via a
  // synthetic `<a download>` click. The token-based URL is unauthenticated;
  // we must build it client-side rather than redirecting.
  const onDownloadInstaller = useCallback(async () => {
    const pkg = softwareTitle?.software_package;
    if (!pkg || typeof teamIdForApi !== "number") return;
    try {
      const resp = await softwareAPI.getSoftwarePackageToken(
        softwareId,
        teamIdForApi
      );
      if (!resp.token) {
        throw new Error("No download token returned");
      }
      const { origin } = global.window.location;
      const url = `${origin}${URL_PREFIX}/api${endpoints.SOFTWARE_PACKAGE_TOKEN(
        softwareId
      )}/${resp.token}`;
      const a = document.createElement("a");
      a.href = url;
      a.download = pkg.name;
      a.click();
      a.remove();
    } catch (e) {
      notify.error("Couldn't download. Please try again.");
    }
  }, [softwareId, softwareTitle, teamIdForApi]);

  const onTeamChange = useCallback(
    (teamId: number) => {
      handleTeamChange(teamId);
    },
    [handleTeamChange]
  );

  const renderSoftwareSummaryCard = (title: ISoftwareTitleDetails) => {
    return (
      <SoftwareSummaryCard
        softwareTitle={title}
        softwareId={softwareId}
        teamId={teamIdForApi}
        router={router}
        refetchSoftwareTitle={refetchSoftwareTitle}
        onToggleViewYaml={onToggleViewYaml}
        onClickVersions={() => setShowVersionsModal(true)}
      />
    );
  };

  const renderLibrarySection = (title: ISoftwareTitleDetails) => {
    // Library section is Premium-only
    // Fleet Free should not see it even when an installer is present.
    if (!isPremiumTier || !isAvailableForInstall) {
      return null;
    }

    const openEditModal = () => setShowLibraryEditModal(true);

    const statusPath = (software_status: "installed" | "pending" | "failed") =>
      getPathWithQueryParams(paths.MANAGE_HOSTS, {
        software_title_id: softwareId,
        software_status,
        fleet_id: currentTeamId ?? APP_CONTEXT_NO_TEAM_ID,
      });

    const libraryAccordionList = () => {
      const pkg = title.software_package;
      const appStore = title.app_store_app;

      if (appStore) {
        const { labels, kind } = pickLabels(appStore);
        const isAndroidPlayStoreApp = appStore.platform === "android";
        const isIosOrIpadosApp = isIpadOrIphoneSoftwareSource(title.source);
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
              isIosOrIpadosApp={isIosOrIpadosApp}
              isActive
              labels={labels}
              labelKind={kind}
              canEditSoftware={canEditSoftware}
              installed={appStore.status?.installed ?? 0}
              pending={appStore.status?.pending ?? 0}
              failed={appStore.status?.failed ?? 0}
              installedPath={statusPath("installed")}
              pendingPath={statusPath("pending")}
              failedPath={statusPath("failed")}
              onLabelCountClick={openEditModal}
              onLabelsClick={openEditModal}
              onTrashClick={() => setShowDeleteModal(true)}
            />
          </LibraryItemAccordionList>
        );
      }

      if (!pkg) return null;
      const { labels, kind } = pickLabels(pkg);
      const isFma = installerResult?.meta.isFleetMaintainedApp ?? false;
      const isLatestFmaVersion =
        installerResult?.meta.isLatestFmaVersion ?? false;
      const isScriptPackage =
        installerResult?.cardInfo.isScriptPackage ?? false;
      const { installed, pending, failed } = aggregateInstallStatusCounts(
        pkg.status
      );
      // FMAs list every cached version (active row badged from the pin, the
      // rest dimmed rollback candidates); other packages render a single row.
      const rows = buildLibraryVersionRows({
        fleetMaintainedVersions: pkg.fleet_maintained_versions,
        activeVersion: pkg.version,
        pinnedVersion: pkg.pinned_version,
        addedTimestamp: pkg.uploaded_at,
      });
      return (
        <LibraryItemAccordionList>
          {rows.map((row) => (
            <LibraryItemAccordion
              key={row.id}
              filename={row.filename ?? pkg.name}
              version={row.version}
              addedAt={row.uploaded_at}
              installerType="package"
              isFma={isFma}
              isLatestFmaVersion={row.isActive && isLatestFmaVersion}
              isScriptPackage={isScriptPackage}
              isTarballPackage={title.source === "tgz_packages"}
              isIosOrIpadosApp={isIpadOrIphoneSoftwareSource(title.source)}
              isActive={row.isActive}
              badgeState={row.badgeState}
              labels={row.isActive ? labels : null}
              labelKind={kind}
              canEditSoftware={canEditSoftware}
              installed={row.isActive ? installed : 0}
              pending={row.isActive ? pending : 0}
              failed={row.isActive ? failed : 0}
              installedPath={statusPath("installed")}
              pendingPath={statusPath("pending")}
              failedPath={statusPath("failed")}
              hashSha256={row.isActive ? pkg.hash_sha256 ?? null : null}
              canDownload={row.isActive}
              onBadgeClick={
                isFma && canEditSoftware
                  ? () => setShowVersionsModal(true)
                  : undefined
              }
              onLabelCountClick={openEditModal}
              onLabelsClick={openEditModal}
              onDownloadClick={onDownloadInstaller}
              onTrashClick={() => setShowDeleteModal(true)}
            />
          ))}
        </LibraryItemAccordionList>
      );
    };

    return (
      <section className={`${baseClass}__section`}>
        <SectionHeader title="Library" />
        <PageDescription content="Software available to be installed" />
        {libraryAccordionList()}
      </section>
    );
  };

  const renderInventorySection = (title: ISoftwareTitleDetails) => {
    // Hide for sources that don't report versions/hosts (tgz/sh/ps1 packages)
    // and when no hosts have the software installed yet.
    const showInventorySection =
      !!title.hosts_count &&
      !NO_VERSION_OR_HOST_DATA_SOURCES.includes(title.source);

    if (!showInventorySection) {
      return null;
    }

    return (
      <section className={`${baseClass}__section`}>
        <SectionHeader title="Inventory" />
        <PageDescription content="Versions installed across all hosts" />
        <TitleVersionsTable
          router={router}
          data={title.versions ?? []}
          isLoading={isSoftwareTitleLoading}
          teamIdForApi={teamIdForApi}
          isIPadOSOrIOSApp={isIpadOrIphoneSoftwareSource(title.source)}
          isAvailableForInstall={isAvailableForInstall}
          countsUpdatedAt={title.counts_updated_at}
        />
      </section>
    );
  };

  // Renders the YAML modal for custom (non-FMA) packages. Two flows open it:
  // (1) `?gitops_yaml=true` redirect after add, and (2) editing a custom
  // package in gitops mode — `EditSoftwareModal` calls `openViewYamlModal()`
  // instead of flashing success.
  const renderViewYamlModal = (title: ISoftwareTitleDetails) => {
    if (!showViewYamlModal) return null;
    const pkg = title.software_package;
    // FMA packages don't expose YAML editing — only custom packages do.
    if (!pkg || pkg.fleet_maintained_app_id) return null;
    return (
      <ViewYamlModal
        softwareTitleName={title.name}
        iconUrl={title.icon_url}
        displayName={getDisplayedSoftwareName(title.name, title.display_name)}
        softwarePackage={pkg}
        onExit={onToggleViewYaml}
        isScriptPackage={installerResult?.cardInfo.isScriptPackage}
      />
    );
  };

  // Delete modal for the active library row's installer.
  const renderDeleteModal = () => {
    if (!showDeleteModal || typeof teamIdForApi !== "number") return null;
    const meta = installerResult?.meta;
    const isAndroidApp = !!meta?.isAndroidPlayStoreApp;
    const isAppStoreApp = meta?.installerType === "app-store" && !isAndroidApp;
    return (
      <DeleteSoftwareModal
        softwareId={softwareId}
        teamId={teamIdForApi}
        gitOpsModeEnabled={gitOpsModeEnabled}
        isAppStoreApp={isAppStoreApp}
        isAndroidApp={isAndroidApp}
        onExit={() => setShowDeleteModal(false)}
        onSuccess={() => {
          setShowDeleteModal(false);
          onDeleteInstaller();
        }}
      />
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

  const renderVersionsModal = (title: ISoftwareTitleDetails) => {
    // `teamIdForApi` is undefined on "All teams" (where `currentTeamId` is the
    // -1 sentinel); guard so we never PATCH `fleet_id=-1`. Mirrors the delete modal.
    if (
      !showVersionsModal ||
      !title.software_package ||
      typeof teamIdForApi !== "number"
    ) {
      return null;
    }
    return (
      <VersionsModal
        softwareTitle={title}
        softwareId={softwareId}
        teamId={teamIdForApi}
        refetchSoftwareTitle={refetchSoftwareTitle}
        onExit={() => setShowVersionsModal(false)}
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
      // Intentional: a title with no installer and no installed hosts collapses
      // to just the summary card with both Library and Inventory hidden. No
      // empty state is shown — design wants the summary alone in that case.
      return (
        <>
          {renderSoftwareSummaryCard(softwareTitle)}
          {renderLibrarySection(softwareTitle)}
          {renderInventorySection(softwareTitle)}
          {renderLibraryEditModal(softwareTitle)}
          {renderDeleteModal()}
          {renderViewYamlModal(softwareTitle)}
          {renderVersionsModal(softwareTitle)}
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
