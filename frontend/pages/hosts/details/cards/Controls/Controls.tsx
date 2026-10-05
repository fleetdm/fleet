import classnames from "classnames";
import React, { useCallback, useContext, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import Slider from "components/forms/fields/Slider";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import { AppContext } from "context/app";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import ControlDetailsModal from "./ControlDetailsModal";
import generateTableConfig, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";
import UninstallProfileModal from "./UninstallProfileModal";

const baseClass = "controls-card";

const profileStatusKey = (profile: IHostMdmProfileWithAddedStatus) =>
  `${profile.status}|${profile.operation_type}`;

const HIDDEN_PROFILES_TOOLTIP =
  "Some of the profiles on this device have been hidden by your IT admin. These include automatically installed profiles that don't require action from you.";

interface IControlsProps {
  /** Rows derived from the host's MDM data by `generateTableData`. */
  controls: IHostMdmProfileWithAddedStatus[];
  hostDisplayName: string;
  /** My device: second person, and no Controls page to link to. */
  isDeviceUser?: boolean;
  isPremiumTier?: boolean;
  /** Fleet setting for macOS: disk encryption enforced without key escrow. */
  isMacOSDiskEncryptionEnforceOnly?: boolean;
  canResendProfiles: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  /** Whether the user can reach the Controls page the empty state links to. */
  canAddControls?: boolean;
  /** Whether the host has an active MDM connection with *this* Fleet. A host
   * managed by a third-party MDM reads as enrolled but still can't receive
   * Fleet controls, so the empty state keys off this, not enrollment status. */
  isConnectedToFleetMdm?: boolean;
  /** Fleet the host belongs to, preserved on the "Add controls" link. */
  teamId?: number | null;
  resendRequest: (profileUUID: string) => Promise<void>;
  resendCertificateRequest?: (certificateTemplateId: number) => Promise<void>;
  rotateRecoveryLockPassword?: () => Promise<void>;
  resendHostNameTemplate?: () => Promise<void>;
  onProfileResent: () => void | Promise<unknown>;
  /** Self-service (opt-in) profiles are macOS-only for now. */
  isMacOSHost?: boolean;
  canManageSelfServiceProfiles?: boolean;
  installRequest?: (profileUUID: string) => Promise<void>;
  uninstallRequest?: (profileUUID: string) => Promise<void>;
  router: InjectedRouter;
  className?: string;
}

const Controls = ({
  controls,
  hostDisplayName,
  isDeviceUser = false,
  isMacOSDiskEncryptionEnforceOnly = false,
  canResendProfiles,
  canRotateRecoveryLockPassword = false,
  canResendHostNameTemplate = false,
  canAddControls = false,
  isConnectedToFleetMdm = false,
  teamId,
  resendRequest,
  resendCertificateRequest,
  rotateRecoveryLockPassword,
  resendHostNameTemplate,
  onProfileResent,
  isMacOSHost = false,
  canManageSelfServiceProfiles = false,
  installRequest,
  uninstallRequest,
  isPremiumTier = false,
  router,
  className,
}: IControlsProps) => {
  const [
    selectedControl,
    setSelectedControl,
  ] = useState<IHostMdmProfileWithAddedStatus | null>(null);
  const [typeFilter, setTypeFilter] = useState<string>("all_available");
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [showHiddenProfiles, setShowHiddenProfiles] = useState(false);
  const [
    profileToUninstall,
    setProfileToUninstall,
  ] = useState<IHostMdmProfileWithAddedStatus | null>(null);
  // Install/uninstall only record the opt-in; the host's profile status
  // changes on the reconciler's next run. Remember the status each request
  // was sent from and keep the row's actions disabled until it changes.
  const [requestedFromStatus, setRequestedFromStatus] = useState<
    Record<string, string>
  >({});
  const { config } = useContext(AppContext);

  // Admin-only. The device page never populates AppContext.config and the
  // server refuses the device-token resend of this profile anyway; the
  // isDeviceUser check just keeps that explicit here.
  const canResendFleetdWhileVerifying =
    !isDeviceUser && !!config?.auth?.mdm_apple_one_time_enroll_secrets;

  const markRequested = useCallback(
    (profile: IHostMdmProfileWithAddedStatus) =>
      setRequestedFromStatus((prev) => ({
        ...prev,
        [profile.profile_uuid]: profileStatusKey(profile),
      })),
    []
  );

  const isActionRequested = useCallback(
    (profile: IHostMdmProfileWithAddedStatus) =>
      requestedFromStatus[profile.profile_uuid] === profileStatusKey(profile),
    [requestedFromStatus]
  );

  const onInstall = useMemo(
    () =>
      installRequest &&
      (async (profile: IHostMdmProfileWithAddedStatus) => {
        await installRequest(profile.profile_uuid);
        markRequested(profile);
        onProfileResent();
      }),
    [installRequest, markRequested, onProfileResent]
  );

  const onClickUninstall = useMemo(
    () => (uninstallRequest ? setProfileToUninstall : undefined),
    [uninstallRequest]
  );

  const tableConfig = useMemo(
    () =>
      generateTableConfig({
        canResendProfiles,
        resendRequest,
        onProfileResent,
        resendCertificateRequest,
        canRotateRecoveryLockPassword,
        rotateRecoveryLockPassword,
        canResendHostNameTemplate,
        resendHostNameTemplate,
        canResendFleetdWhileVerifying,
        canManageSelfServiceProfiles,
        onInstall,
        onClickUninstall,
        isActionRequested,
        isDeviceUser,
      }),
    [
      canResendProfiles,
      resendRequest,
      onProfileResent,
      resendCertificateRequest,
      canRotateRecoveryLockPassword,
      rotateRecoveryLockPassword,
      canResendHostNameTemplate,
      resendHostNameTemplate,
      canResendFleetdWhileVerifying,
      canManageSelfServiceProfiles,
      onInstall,
      onClickUninstall,
      isActionRequested,
      isDeviceUser,
    ]
  );

  const onUninstalled = useCallback(
    (profileUUID: string) => {
      const profile = controls.find((c) => c.profile_uuid === profileUUID);
      if (profile) markRequested(profile);
      onProfileResent();
    },
    [controls, markRequested, onProfileResent]
  );

  const onResentFromModal = useCallback(() => {
    setSelectedControl(null);
    onProfileResent();
  }, [onProfileResent]);

  const emptyStateInfo = () => {
    if (!isConnectedToFleetMdm) {
      return isDeviceUser
        ? "No controls available. Your device isn't talking to Fleet for MDM features."
        : "No controls available. This host isn't talking to Fleet for MDM features.";
    }
    return isDeviceUser
      ? "No controls have been added for your device."
      : "No controls have been added for this host.";
  };

  const onAddControls = useCallback(() => {
    // A no-team host reports `team_id: null`, but the Controls page reads "No
    // team" as fleet_id=0 — an absent param means "keep whatever fleet you were
    // on", which lands the user somewhere else entirely.
    router.push(
      getPathWithQueryParams(PATHS.CONTROLS, { fleet_id: teamId ?? 0 })
    );
  }, [router, teamId]);

  const filteredControls = useMemo(() => {
    let filtered = controls;
    if (isDeviceUser && !showHiddenProfiles) {
      filtered = filtered.filter((control) => !control.hidden);
    }
    if (typeFilter === "self_service") {
      filtered = filtered.filter((control) => control.self_service);
    }
    if (searchQuery) {
      filtered = filtered.filter((control) =>
        control.name.toLowerCase().includes(searchQuery.toLowerCase())
      );
    }
    return filtered;
  }, [controls, isDeviceUser, showHiddenProfiles, typeFilter, searchQuery]);

  const tableCustomFilters = (): JSX.Element | null => {
    if (!isMacOSHost) {
      return null;
    }
    if (isDeviceUser) {
      if (!isPremiumTier) {
        return null;
      }

      return (
        <Slider
          value={showHiddenProfiles}
          onChange={() => setShowHiddenProfiles(!showHiddenProfiles)}
          activeText="Show hidden profiles"
          inactiveText="Show hidden profiles"
          ariaLabel="Show hidden profiles"
          labelTooltip={HIDDEN_PROFILES_TOOLTIP}
        />
      );
    }
    return (
      <>
        <DropdownWrapper
          className={`${baseClass}__filter`}
          variant="table-filter"
          name="type-filter"
          onChange={(value) => {
            if (value?.value) {
              setTypeFilter(value.value);
            }
          }}
          options={[
            {
              label: "All available",
              value: "all_available",
              helpText: "Profiles that can be installed on this host.",
            },
            {
              label: "End user initiated (manual)",
              value: "self_service",
              helpText: (
                <span>
                  Profiles that end users can opt-in to from{" "}
                  <strong>Fleet Desktop &gt; Controls</strong>.
                </span>
              ),
            },
          ]}
          value={typeFilter}
        />
      </>
    );
  };

  return (
    <div
      className={classnames(baseClass, className, {
        [`${baseClass}--self-service`]: canManageSelfServiceProfiles,
      })}
    >
      <TableContainer
        columnConfigs={tableConfig}
        data={filteredControls}
        isLoading={false}
        resultsTitle="controls"
        defaultSortHeader="status"
        defaultSortDirection="asc"
        renderCount={() => (
          <TableCount name="controls" count={filteredControls.length} />
        )}
        emptyComponent={() => (
          <EmptyState
            header="No controls"
            info={emptyStateInfo()}
            primaryButton={
              // Adding controls doesn't help a host that can't receive them.
              !isDeviceUser && canAddControls && isConnectedToFleetMdm ? (
                <Button onClick={onAddControls} type="button">
                  Add controls
                </Button>
              ) : undefined
            }
          />
        )}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        disableMultiRowSelect // Removes the multi-select checkbox column
        // Not-installed opt-in profiles have no status to show details for.
        canClickRow={(row: Row<IHostMdmProfileWithAddedStatus>) =>
          row.original.status !== null
        }
        onClickRow={(row: Row<IHostMdmProfileWithAddedStatus>) => {
          setSelectedControl(row.original);
        }}
        inputPlaceHolder="Search by name"
        // My device only gets the hidden-profiles toggle.
        searchable={!isDeviceUser}
        searchQuery={searchQuery}
        onQueryChange={(data) => {
          setSearchQuery(data.searchQuery);
        }}
        isClientSideFilter
        customControl={tableCustomFilters}
        keyboardSelectableRows
        isClientSidePagination
      />
      {selectedControl && (
        <ControlDetailsModal
          control={selectedControl}
          hostDisplayName={hostDisplayName}
          isDeviceUser={isDeviceUser}
          isMacOSDiskEncryptionEnforceOnly={isMacOSDiskEncryptionEnforceOnly}
          canResendProfiles={canResendProfiles}
          canResendFleetdWhileVerifying={canResendFleetdWhileVerifying}
          canRotateRecoveryLockPassword={canRotateRecoveryLockPassword}
          canResendHostNameTemplate={canResendHostNameTemplate}
          resendRequest={resendRequest}
          resendCertificateRequest={resendCertificateRequest}
          rotateRecoveryLockPassword={rotateRecoveryLockPassword}
          resendHostNameTemplate={resendHostNameTemplate}
          onProfileResent={onResentFromModal}
          onExit={() => setSelectedControl(null)}
        />
      )}
      {profileToUninstall && uninstallRequest && (
        <UninstallProfileModal
          profileName={profileToUninstall.name}
          profileUUID={profileToUninstall.profile_uuid}
          hostDisplayName={hostDisplayName}
          isDeviceUser={isDeviceUser}
          uninstallRequest={uninstallRequest}
          onSuccess={onUninstalled}
          onExit={() => setProfileToUninstall(null)}
        />
      )}
    </div>
  );
};

export default Controls;
