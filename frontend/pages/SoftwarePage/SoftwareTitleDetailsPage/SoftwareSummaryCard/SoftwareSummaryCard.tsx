/** software/titles/:id > First section */

import React, { useContext, useMemo, useState } from "react";

import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { pluralize } from "utilities/strings/stringUtils";
import { AppContext } from "context/app";
import { useSoftwareInstaller } from "hooks/useSoftwareInstallerMeta";
import {
  formatSoftwareType,
  isAndroidSoftwareSource,
  isIpadOrIphoneSoftwareSource,
  ISoftwareTitleDetails,
} from "interfaces/software";

import {
  getDisplayedSoftwareName,
  getSelfServiceTooltip,
  mergePolicies,
} from "pages/SoftwarePage/helpers";
import Card from "components/Card";
import Chip from "components/Chip";
import SoftwareDetailsSummary from "pages/SoftwarePage/components/cards/SoftwareDetailsSummary";
import EditIconModal from "../EditIconModal";
import EditSoftwareModal from "../EditSoftwareModal";
import EditConfigurationModal from "../EditConfigurationModal";
import EditAutoUpdateConfigModal from "../EditAutoUpdateConfigModal";
import AddPatchPolicyModal from "../AddPatchPolicyModal";
import PoliciesModal from "../PoliciesModal";

interface ISoftwareSummaryCard {
  softwareTitle: ISoftwareTitleDetails;
  softwareId: number;
  teamId?: number;
  router: InjectedRouter;
  refetchSoftwareTitle: () => void;
  /** Opens the page-owned Versions modal; the Actions item is gated here by
   * `canManageVersions`. */
  onClickVersions: () => void;
  /** Canonical "this title can hold multiple custom packages" flag. When
   * true the card hides its Self-service / Auto install / Patch chips
   * (Library accordion rows show per-package icons instead) and collapses
   * the Actions dropdown into a single pencil-icon Edit-appearance button. */
  canActivateMultiplePackages?: boolean;
}

const baseClass = "software-summary-card";

const getPolicyChipTooltip = (
  isPatchPolicyOnly: boolean,
  isSinglePolicy: boolean
) => {
  if (isPatchPolicyOnly) {
    return <>Policy fails if the host is on an older version.</>;
  }
  return isSinglePolicy ? (
    <>Policy triggers install.</>
  ) : (
    <>Policies trigger install.</>
  );
};

const SoftwareSummaryCard = ({
  softwareTitle,
  softwareId,
  teamId,
  router,
  refetchSoftwareTitle,
  onClickVersions,
  canActivateMultiplePackages = false,
}: ISoftwareSummaryCard) => {
  const { isPremiumTier } = useContext(AppContext);
  const installerResult = useSoftwareInstaller(softwareTitle);

  const [iconUploadedAt, setIconUploadedAt] = useState("");
  const [showEditIconModal, setShowEditIconModal] = useState(false);
  const [showEditSoftwareModal, setShowEditSoftwareModal] = useState(false);
  const [showAddPatchPolicyModal, setShowAddPatchPolicyModal] = useState(false);
  const [showEditConfigurationModal, setShowEditConfigurationModal] = useState(
    false
  );
  const [
    showEditAutoUpdateConfigModal,
    setShowEditAutoUpdateConfigModal,
  ] = useState(false);
  const [showPoliciesModal, setShowPoliciesModal] = useState(false);

  const softwareDisplayName = getDisplayedSoftwareName(
    softwareTitle.name,
    softwareTitle.display_name
  );

  // Pre-compute meta-derived values via optional chaining so the hooks below
  // can run unconditionally (React requires stable hook order across renders).
  const installerType = installerResult?.meta.installerType;
  const isFleetMaintainedApp = !!installerResult?.meta.isFleetMaintainedApp;
  const isAndroidPlayStoreApp = !!installerResult?.meta.isAndroidPlayStoreApp;
  const isCustomPackage = !!installerResult?.meta.isCustomPackage;

  // Depend on the optional-chained sources directly so the memo's cache hits
  // when both are nullish — `?? []` would mint a fresh array literal each
  // render and bust the cache (and cascade into `headerPills` below).
  const packageAutoInstallPolicies =
    softwareTitle.software_package?.automatic_install_policies;
  const appStoreAutoInstallPolicies =
    softwareTitle.app_store_app?.automatic_install_policies;
  const patchPolicy = softwareTitle.software_package?.patch_policy;
  const mergedPolicies = useMemo(
    () =>
      mergePolicies({
        automaticInstallPolicies:
          packageAutoInstallPolicies ?? appStoreAutoInstallPolicies,
        patchPolicy,
      }),
    [packageAutoInstallPolicies, appStoreAutoInstallPolicies, patchPolicy]
  );
  const isSelfService =
    !!softwareTitle.software_package?.self_service ||
    !!softwareTitle.app_store_app?.self_service;
  // Show the Auto install pill whenever the title has any linked policy —
  // patch-only policies live under `software_package.patch_policy` (not in
  // `automatic_install_policies`), so we key off the merged set.
  const hasLinkedPolicies = mergedPolicies.length > 0;
  // If every linked policy is patch-only, surface a "Patch policy" pill
  // instead of "Auto install". When at least one policy is dynamic, the
  // Auto install pill wins (it's the stronger statement).
  const isPatchPolicyOnly =
    hasLinkedPolicies && mergedPolicies.every((p) => !p.type.has("dynamic"));

  // VPP / App Store distinguishes by installer source (app-store vs package),
  // not by the host OS — VPP apps can be macOS too, and an iOS/iPadOS title can
  // ship as a custom package.
  const isAppleVpp = installerType === "app-store" && !isAndroidPlayStoreApp;
  // Multi-package custom titles pluralize the kind chip — a title with two
  // .pkg installers reads "Custom packages", not "Custom package". Falls back
  // to singular for single-package titles (and back-compat responses where
  // `packages` is still null — `pluralize(0)` also returns the plural form,
  // but those titles never reach the custom-package branch of `find` below).
  const customPackageCount = softwareTitle.packages?.length ?? 1;
  const customPackageChipLabel = pluralize(
    customPackageCount,
    "Custom package"
  );
  // Order matters: `.find` returns the first truthy row. FMA is checked first
  // because a Fleet-maintained app uploaded as a custom package still counts
  // as FMA; Apple VPP precedes Play Store so cross-platform store titles label
  // by their dominant source; Custom package is the catch-all fallback.
  const installerKindLabel = ([
    [isFleetMaintainedApp, "Fleet-maintained"],
    [isAppleVpp, "App Store (VPP)"],
    [isAndroidPlayStoreApp, "Play Store"],
    [isCustomPackage, customPackageChipLabel],
  ] as const).find(([flag]) => flag)?.[1];

  // Titles that can hold multiple custom packages move Self-service and
  // Auto-install/Patch indicators down to per-row icons on the Library
  // accordion. The title-level chips would be misleading when one package
  // is self-service and another isn't. FMA and iOS in-house .ipa keep the
  // chips since they're single-package — the flag is owned by the page.
  const showSelfServiceChip = isSelfService && !canActivateMultiplePackages;
  const showAutoInstallChip = hasLinkedPolicies && !canActivateMultiplePackages;

  const showHeaderPills =
    !!installerKindLabel || showSelfServiceChip || showAutoInstallChip;

  const headerPills = useMemo(() => {
    if (!showHeaderPills) {
      return undefined;
    }
    return (
      <>
        {installerKindLabel && <Chip text={installerKindLabel} />}
        {showSelfServiceChip && (
          <Chip
            icon="user"
            text="Self-service"
            tooltip={getSelfServiceTooltip(
              isIpadOrIphoneSoftwareSource(softwareTitle.source),
              isAndroidSoftwareSource(softwareTitle.source)
            )}
          />
        )}
        {showAutoInstallChip && (
          <Chip
            icon={isPatchPolicyOnly ? undefined : "refresh"}
            text={isPatchPolicyOnly ? "Patch policy" : "Auto install"}
            onClick={() => {
              // Single-policy case: jump straight to the policy. The modal
              // would just show a one-item list with that same link.
              if (mergedPolicies.length === 1) {
                router.push(
                  getPathWithQueryParams(
                    PATHS.POLICY_DETAILS(mergedPolicies[0].id),
                    { fleet_id: teamId }
                  )
                );
                return;
              }
              setShowPoliciesModal(true);
            }}
            tooltip={getPolicyChipTooltip(
              isPatchPolicyOnly,
              mergedPolicies.length === 1
            )}
          />
        )}
      </>
    );
  }, [
    showHeaderPills,
    installerKindLabel,
    showSelfServiceChip,
    showAutoInstallChip,
    isPatchPolicyOnly,
    mergedPolicies,
    softwareTitle.source,
    router,
    teamId,
  ]);

  const policiesModal = showPoliciesModal && (
    <PoliciesModal
      policies={mergedPolicies}
      teamId={teamId}
      onExit={() => setShowPoliciesModal(false)}
    />
  );

  if (!installerResult) {
    return (
      <>
        <Card borderRadiusSize="xxlarge" className={baseClass}>
          <SoftwareDetailsSummary
            displayName={softwareDisplayName}
            type={formatSoftwareType(softwareTitle)}
            versions={softwareTitle.versions?.length ?? 0}
            hostCount={softwareTitle.hosts_count}
            countsUpdatedAt={softwareTitle.counts_updated_at}
            queryParams={{ software_title_id: softwareId, fleet_id: teamId }}
            name={softwareTitle.name}
            source={softwareTitle.source}
            iconUrl={softwareTitle.icon_url}
            iconUploadedAt={iconUploadedAt}
            headerPills={headerPills}
          />
        </Card>
        {policiesModal}
      </>
    );
  }

  const {
    softwareInstaller,
    isIosOrIpadosApp,
    isAndroidPlayStoreWebApp,
    canManageSoftware,
  } = installerResult.meta;

  const canEditAppearance = canManageSoftware;
  const canEditSoftware = canManageSoftware && !isAndroidPlayStoreApp;
  /** Permission to manage software + Google Playstore app (not a web app) or iOS/iPadOS app */
  const canEditConfiguration =
    canManageSoftware &&
    ((isAndroidPlayStoreApp && !isAndroidPlayStoreWebApp) || isIosOrIpadosApp);
  const canPatchSoftware = canManageSoftware && isFleetMaintainedApp;
  /** Versions / pin is a Premium-only Fleet-maintained app feature */
  const canManageVersions =
    canManageSoftware && isFleetMaintainedApp && !!isPremiumTier;
  /** Installer modals require a specific team; hidden from "All Teams" */
  const hasValidTeamId = typeof teamId === "number" && teamId >= 0;
  const softwareInstallerOnTeam = hasValidTeamId && softwareInstaller;

  const canEditAutoUpdateConfig =
    softwareTitle.app_store_app && isIosOrIpadosApp && canManageSoftware;

  const onClickEditAppearance = () => setShowEditIconModal(true);
  const onClickEditSoftware = () => setShowEditSoftwareModal(true);
  const onClickAddPatchPolicy = () => setShowAddPatchPolicyModal(true);
  const onClickEditConfiguration = () => setShowEditConfigurationModal(true);
  const onClickEditAutoUpdateConfig = () =>
    setShowEditAutoUpdateConfigModal(true);

  return (
    <>
      <Card borderRadiusSize="xxlarge" className={baseClass}>
        <SoftwareDetailsSummary
          displayName={softwareDisplayName}
          type={formatSoftwareType(softwareTitle)}
          versions={softwareTitle.versions?.length ?? 0}
          hostCount={softwareTitle.hosts_count}
          countsUpdatedAt={softwareTitle.counts_updated_at}
          queryParams={{
            software_title_id: softwareId,
            fleet_id: teamId,
          }}
          name={softwareTitle.name}
          source={softwareTitle.source}
          iconUrl={softwareTitle.icon_url}
          iconUploadedAt={iconUploadedAt}
          canManageSoftware={canManageSoftware}
          onClickEditAppearance={
            canEditAppearance ? onClickEditAppearance : undefined
          }
          onClickEditSoftware={
            // Multi-package titles move per-installer editing to the Library
            // accordion row; the page-level Edit button collapses to a single
            // pencil-icon Edit-appearance button below. Single-package types
            // (FMA, VPP, Google Play, iOS in-house .ipa) keep the Actions
            // dropdown.
            canEditSoftware && !canActivateMultiplePackages
              ? onClickEditSoftware
              : undefined
          }
          useSingleEditAppearanceButton={canActivateMultiplePackages}
          onClickAddPatchPolicy={
            canPatchSoftware ? onClickAddPatchPolicy : undefined
          }
          onClickVersions={canManageVersions ? onClickVersions : undefined}
          onClickEditConfiguration={
            canEditConfiguration ? onClickEditConfiguration : undefined
          }
          onClickEditAutoUpdateConfig={
            canEditAutoUpdateConfig ? onClickEditAutoUpdateConfig : undefined
          }
          patchPolicyId={softwareTitle.software_package?.patch_policy?.id}
          headerPills={headerPills}
          isAppleVpp={isAppleVpp}
        />
      </Card>
      {showEditIconModal && softwareInstallerOnTeam && (
        <EditIconModal
          softwareId={softwareId}
          teamIdForApi={teamId}
          software={softwareInstaller}
          onExit={() => setShowEditIconModal(false)}
          refetchSoftwareTitle={refetchSoftwareTitle}
          iconUploadedAt={iconUploadedAt}
          setIconUploadedAt={setIconUploadedAt}
          installerType={installerResult.meta.installerType}
          previewInfo={{
            name: softwareDisplayName,
            titleName: softwareTitle.name,
            type: formatSoftwareType(softwareTitle),
            source: softwareTitle.source,
            currentIconUrl: softwareTitle.icon_url,
            versions: softwareTitle.versions?.length ?? 0,
            countsUpdatedAt: softwareTitle.counts_updated_at,
            selfServiceVersion: softwareInstaller.version,
          }}
        />
      )}
      {showEditSoftwareModal && softwareInstallerOnTeam && (
        <EditSoftwareModal
          softwareId={softwareId}
          teamId={teamId}
          softwareInstaller={softwareInstaller}
          onExit={() => setShowEditSoftwareModal(false)}
          refetchSoftwareTitle={refetchSoftwareTitle}
          installerType={installerResult.meta.installerType}
          isFleetMaintainedApp={isFleetMaintainedApp}
          isIosOrIpadosApp={isIosOrIpadosApp}
          name={softwareTitle.name}
          displayName={softwareDisplayName}
          source={softwareTitle.source}
          iconUrl={softwareTitle.icon_url}
        />
      )}
      {showAddPatchPolicyModal && softwareInstallerOnTeam && (
        <AddPatchPolicyModal
          softwareId={softwareTitle.id}
          teamId={teamId}
          onSuccess={refetchSoftwareTitle}
          onExit={() => setShowAddPatchPolicyModal(false)}
        />
      )}
      {showEditConfigurationModal && softwareInstallerOnTeam && (
        <EditConfigurationModal
          softwareInstaller={softwareInstaller}
          softwareId={softwareId}
          teamId={teamId}
          isApplePlatform={isIosOrIpadosApp}
          refetchSoftwareTitle={refetchSoftwareTitle}
          onExit={() => setShowEditConfigurationModal(false)}
        />
      )}
      {showEditAutoUpdateConfigModal && softwareInstallerOnTeam && (
        <EditAutoUpdateConfigModal
          softwareTitle={softwareTitle}
          teamId={teamId}
          refetchSoftwareTitle={refetchSoftwareTitle}
          onExit={() => setShowEditAutoUpdateConfigModal(false)}
        />
      )}
      {policiesModal}
    </>
  );
};

export default SoftwareSummaryCard;
