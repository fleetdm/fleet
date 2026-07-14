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
  ISoftwareInstallPolicyUI,
  ISoftwarePackage,
  ISoftwareTitleDetails,
  MAX_PACKAGES_PER_TITLE,
  NO_VERSION_OR_HOST_DATA_SOURCES,
} from "interfaces/software";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import {
  canDownloadSoftwareInstaller,
  canWriteSoftware,
} from "utilities/permissions/permissions";
import softwareAPI, {
  ISoftwareTitleResponse,
  IGetSoftwareTitleQueryKey,
} from "services/entities/software";

import { getPathWithQueryParams } from "utilities/url";
import endpoints from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { notify } from "components/ToastNotification";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
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
import AddPackageModal from "./AddPackageModal";
import PoliciesModal from "./PoliciesModal";
import VersionsModal from "./VersionsModal";
import { getDisplayedSoftwareName, mergePolicies } from "../helpers";
import { buildLibraryVersionRows, canDownloadInstallerRow } from "./helpers";
import TitleVersionsTable from "./TitleVersionsTable";

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
  const canDownloadInstaller = canDownloadSoftwareInstaller(
    currentUser,
    currentTeamId ?? null
  );

  const [showLibraryEditModal, setShowLibraryEditModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showAddPackageModal, setShowAddPackageModal] = useState(false);
  // When set, opens a PoliciesModal scoped to a single package's policies
  // (the auto-install icon on a custom-package row). Distinct from the
  // SoftwareSummaryCard's title-aggregate PoliciesModal — that one stays
  // owned by the card and shows policies across all packages.
  const [selectedPackagePolicies, setSelectedPackagePolicies] = useState<
    ISoftwareInstallPolicyUI[] | null
  >(null);
  // Per-installer target for the currently-open Edit or Delete modal on a
  // multi-package title. `null` means "fall back to first-added", which keeps
  // single-package back-compat callers (and the page-level header Edit)
  // pointing at `software_package` without any extra wiring.
  const [selectedInstallerId, setSelectedInstallerId] = useState<number | null>(
    null
  );
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

  // Canonical "this title can hold multiple custom packages" flag.
  // Single source of truth for three coordinated behaviors:
  //   1. Library section shows the "+ Add package" action
  //   2. Accordion rows render the self-service / auto-install icons
  //   3. SoftwareSummaryCard hides the Self-service / Auto install chips
  //      AND collapses the Actions dropdown into a pencil-icon Edit button
  // True for custom non-FMA, non-iOS titles (Mac .pkg, Linux .deb/.rpm/
  // .tar.gz, Windows .msi/.exe, script-only .sh/.ps1). FMA, VPP, Google
  // Play, and iOS in-house .ipa stay single-package. Premium-only.
  const canActivateMultiplePackages =
    !!isPremiumTier &&
    !!installerResult?.meta.isCustomPackage &&
    !installerResult.meta.isIosOrIpadosApp;

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
        onClickVersions={() => setShowVersionsModal(true)}
        canActivateMultiplePackages={canActivateMultiplePackages}
      />
    );
  };

  const renderLibrarySection = (title: ISoftwareTitleDetails) => {
    // Library section is Premium-only
    // Fleet Free should not see it even when an installer is present.
    if (!isPremiumTier || !isAvailableForInstall) {
      return null;
    }

    // `packages` is the source of truth. The `software_package` alias is
    // still returned by the API (points at `packages[0]`, first-added) — the
    // fallback here is defense against a title with no custom packages (e.g.
    // FMA / app-store branch) so downstream code can treat this as an array.
    const packages =
      title.packages ??
      (title.software_package ? [title.software_package] : []);
    const appStore = title.app_store_app;

    // No installable to render at all.
    if (packages.length === 0 && !appStore) {
      return null;
    }

    // Per-row callbacks set `selectedInstallerId` so the modals target the
    // right package on a multi-package title. Called without an id (e.g. from
    // the app-store branch or a back-compat single-package title), they fall
    // back to first-added — same as legacy behavior.
    const openEditModal = (id?: number) => {
      setSelectedInstallerId(id ?? null);
      setShowLibraryEditModal(true);
    };
    const openDeleteModal = (id?: number) => {
      setSelectedInstallerId(id ?? null);
      setShowDeleteModal(true);
    };

    const statusPath = (software_status: "installed" | "pending" | "failed") =>
      getPathWithQueryParams(paths.MANAGE_HOSTS, {
        software_title_id: softwareId,
        software_status,
        fleet_id: currentTeamId ?? APP_CONTEXT_NO_TEAM_ID,
      });

    const renderAppStoreRow = () => {
      if (!appStore) return null;
      const { labels, kind } = pickLabels(appStore);
      const isAndroidPlayStoreApp = appStore.platform === "android";
      const isIosOrIpadosApp = isIpadOrIphoneSoftwareSource(title.source);
      return (
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
          onLabelCountClick={openEditModal}
          onLabelsClick={openEditModal}
          onTrashClick={openDeleteModal}
        />
      );
    };

    // FMAs expand a single package into one badged "active" row plus dimmed
    // rollback rows for every cached version. Custom packages render exactly
    // one row each. With multi-package titles we run this per `pkg` so each
    // top-level entry stays addressable by its own `installer_id`.
    const renderPackageRows = (pkg: ISoftwarePackage) => {
      if (!pkg) return null;
      const { labels, kind } = pickLabels(pkg);
      const isFma = installerResult?.meta.isFleetMaintainedApp ?? false;
      const isLatestFmaVersion =
        installerResult?.meta.isLatestFmaVersion ?? false;
      const isScriptPackage =
        installerResult?.cardInfo.isScriptPackage ?? false;
      const isIosOrIpadosApp = isIpadOrIphoneSoftwareSource(title.source);
      const perPackagePolicies = mergePolicies({
        automaticInstallPolicies: pkg.automatic_install_policies,
        patchPolicy: pkg.patch_policy,
      });
      const hasAutoInstallPolicy = perPackagePolicies.length > 0;
      const { installed, pending, failed } = aggregateInstallStatusCounts(
        pkg.status
      );
      const rows = buildLibraryVersionRows({
        fleetMaintainedVersions: pkg.fleet_maintained_versions,
        activeVersion: pkg.version,
        pinnedVersion: pkg.pinned_version,
        addedTimestamp: pkg.uploaded_at,
      });
      return rows.map((row) => (
        <LibraryItemAccordion
          key={`${pkg.installer_id}-${row.id}`}
          filename={row.filename ?? pkg.name}
          version={row.version}
          addedAt={row.uploaded_at}
          installerType="package"
          isFma={isFma}
          isLatestFmaVersion={row.isActive && isLatestFmaVersion}
          isScriptPackage={isScriptPackage}
          isTarballPackage={title.source === "tgz_packages"}
          isIosOrIpadosApp={isIosOrIpadosApp}
          isActive={row.isActive}
          badgeState={row.badgeState}
          canActivateMultiplePackages={canActivateMultiplePackages}
          isSelfService={pkg.self_service}
          hasAutoInstallPolicy={hasAutoInstallPolicy}
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
          canDownload={canDownloadInstallerRow(
            row.isActive,
            canDownloadInstaller
          )}
          onBadgeClick={
            isFma && canEditSoftware
              ? () => setShowVersionsModal(true)
              : undefined
          }
          onLabelCountClick={() => openEditModal(pkg.installer_id)}
          onLabelsClick={() => openEditModal(pkg.installer_id)}
          onDownloadClick={onDownloadInstaller}
          onTrashClick={() => openDeleteModal(pkg.installer_id)}
          onSelfServiceClick={() => openEditModal(pkg.installer_id)}
          onAutoInstallClick={() => {
            // Single linked policy: jump straight to it (mirrors the chip's
            // "Select to open policy" shortcut). Multiple: open the modal
            // scoped to this specific package, not the aggregate.
            if (perPackagePolicies.length === 1) {
              router.push(
                getPathWithQueryParams(
                  paths.POLICY_DETAILS(perPackagePolicies[0].id),
                  { fleet_id: teamIdForApi }
                )
              );
              return;
            }
            setSelectedPackagePolicies(perPackagePolicies);
          }}
        />
      ));
    };

    // "Add package" lives on titles that can hold multiple custom packages —
    // gated by the page-level `canActivateMultiplePackages` flag. FMA, VPP,
    // Google Play, and iOS in-house .ipa are all single-package by design
    // and so don't surface the action at all. The Library section already
    // early-returns when there are no packages and no app-store app, so we
    // don't need to re-check `packages.length` here.
    const showAddPackageAction = canActivateMultiplePackages && canEditSoftware;
    const atPackageLimit = packages.length >= MAX_PACKAGES_PER_TITLE;
    const addPackageButton = showAddPackageAction && (
      <Button
        variant="inverse"
        onClick={() => setShowAddPackageModal(true)}
        disabled={atPackageLimit}
      >
        <Icon name="plus" />
        Add package
      </Button>
    );
    const headerAction =
      showAddPackageAction && atPackageLimit ? (
        <TooltipWrapper
          tipContent={
            <>
              This title already has {MAX_PACKAGES_PER_TITLE} packages.
              <br />
              Delete one you no longer use before adding.
            </>
          }
          showArrow
          position="left"
          underline={false}
        >
          {addPackageButton}
        </TooltipWrapper>
      ) : (
        addPackageButton
      );

    // App-store and custom-package paths are mutually exclusive at the data
    // layer (the backend rejects custom uploads against an FMA/VPP title), so
    // only one branch ever renders rows. The wrapper stays the same shape.
    // The "Add package" action sits on the description row (not the section
    // header) so it visually aligns with the secondary copy rather than the
    // h2 title — matches the Library row layout in Figma page 2:130.
    return (
      <section className={`${baseClass}__section`}>
        <SectionHeader title="Library" />
        <div className={`${baseClass}__library-description-row`}>
          {/* The multi-package copy is an action prompt — only meaningful to
              a user who can both edit software AND is on a multi-package-
              eligible title. Read-only users and single-package types (FMA,
              VPP, Google Play, iOS in-house .ipa) get the legacy
              "available to be installed" wording. */}
          <PageDescription
            content={
              canActivateMultiplePackages && canEditSoftware
                ? "Add packages for a staged rollout or to support multiple architectures."
                : "Software available to be installed"
            }
          />
          {headerAction}
        </div>
        <LibraryItemAccordionList>
          {/* Row order = API response order. The API returns `packages[]`
              sorted by `installer_id` ascending, so the top row is the
              first-added package (smallest id = collision fallback). The
              UI does not re-sort. */}
          {appStore ? renderAppStoreRow() : packages.map(renderPackageRows)}
        </LibraryItemAccordionList>
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

  // Resolves the targeted package on a multi-package title. Returns the
  // package matching `selectedInstallerId`, or `null` if none matches — in
  // which case the caller falls back to the legacy `software_package` flow.
  const findSelectedPackage = (
    title: ISoftwareTitleDetails
  ): ISoftwarePackage | null => {
    if (selectedInstallerId === null) return null;
    return (
      title.packages?.find((p) => p.installer_id === selectedInstallerId) ??
      null
    );
  };

  const closeDeleteModal = () => {
    setShowDeleteModal(false);
    setSelectedInstallerId(null);
  };

  const closeLibraryEditModal = () => {
    setShowLibraryEditModal(false);
    setSelectedInstallerId(null);
  };

  // Delete modal for the active library row's installer.
  const renderDeleteModal = (title: ISoftwareTitleDetails) => {
    if (!showDeleteModal || typeof teamIdForApi !== "number") return null;
    const meta = installerResult?.meta;
    const isAndroidApp = !!meta?.isAndroidPlayStoreApp;
    const isAppStoreApp = meta?.installerType === "app-store" && !isAndroidApp;
    const selected = findSelectedPackage(title);
    return (
      <DeleteSoftwareModal
        softwareId={softwareId}
        teamId={teamIdForApi}
        installerId={selected?.installer_id}
        gitOpsModeEnabled={gitOpsModeEnabled}
        isAppStoreApp={isAppStoreApp}
        isAndroidApp={isAndroidApp}
        canActivateMultiplePackages={canActivateMultiplePackages}
        onExit={closeDeleteModal}
        onSuccess={() => {
          closeDeleteModal();
          onDeleteInstaller();
        }}
      />
    );
  };

  const renderLibraryEditModal = (title: ISoftwareTitleDetails) => {
    if (!showLibraryEditModal || !installerResult) return null;
    const { meta } = installerResult;
    // On a multi-package title, the row callback set `selectedInstallerId`;
    // resolve it to the actual package so the modal edits the right one.
    // Otherwise (single-package back-compat or app-store), `meta.softwareInstaller`
    // already points at the only installer.
    const selected = findSelectedPackage(title);
    return (
      <EditSoftwareModal
        softwareId={softwareId}
        teamId={currentTeamId ?? APP_CONTEXT_NO_TEAM_ID}
        installerId={selected?.installer_id}
        softwareInstaller={selected ?? meta.softwareInstaller}
        refetchSoftwareTitle={refetchSoftwareTitle}
        onExit={closeLibraryEditModal}
        installerType={meta.installerType}
        isFleetMaintainedApp={meta.isFleetMaintainedApp}
        isIosOrIpadosApp={meta.isIosOrIpadosApp}
        name={title.name}
        displayName={getDisplayedSoftwareName(title.name, title.display_name)}
        source={title.source}
        iconUrl={title.icon_url}
        canActivateMultiplePackages={canActivateMultiplePackages}
      />
    );
  };

  const renderPackagePoliciesModal = () => {
    if (!selectedPackagePolicies) return null;
    return (
      <PoliciesModal
        policies={selectedPackagePolicies}
        teamId={teamIdForApi}
        onExit={() => setSelectedPackagePolicies(null)}
      />
    );
  };

  const renderAddPackageModal = (title: ISoftwareTitleDetails) => {
    if (!showAddPackageModal || typeof teamIdForApi !== "number") return null;
    // First-added is the canonical source for the file-type restriction.
    // Multi-package titles always have `packages[0]`; back-compat titles fall
    // back to `software_package`. The "+ Add package" button is gated on the
    // section being visible, so we always have a name here.
    const existingPackageName =
      title.packages?.[0]?.name ?? title.software_package?.name ?? "";
    return (
      <AddPackageModal
        softwareTitleId={softwareId}
        teamId={teamIdForApi}
        existingPackageName={existingPackageName}
        onExit={() => setShowAddPackageModal(false)}
        onSuccess={() => {
          setShowAddPackageModal(false);
          refetchSoftwareTitle();
        }}
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
          {renderDeleteModal(softwareTitle)}
          {renderAddPackageModal(softwareTitle)}
          {renderPackagePoliciesModal()}
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
