import React, { useCallback, useMemo, useState } from "react";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";
import classnames from "classnames";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import ControlDetailsModal from "./ControlDetailsModal";
import generateTableConfig, {
  IHostMdmProfileWithAddedStatus,
} from "./OSSettingsTableConfig";

const baseClass = "controls-card";

interface IControlsProps {
  /** Rows derived from the host's MDM data by `generateTableData`. */
  controls: IHostMdmProfileWithAddedStatus[];
  hostDisplayName: string;
  /** My device: second person, and no Controls page to link to. */
  isDeviceUser?: boolean;
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
  onProfileResent: () => void;
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
  router,
  className,
}: IControlsProps) => {
  const [
    selectedControl,
    setSelectedControl,
  ] = useState<IHostMdmProfileWithAddedStatus | null>(null);

  const tableConfig = useMemo(
    () =>
      generateTableConfig(
        canResendProfiles,
        resendRequest,
        onProfileResent,
        resendCertificateRequest,
        canRotateRecoveryLockPassword,
        rotateRecoveryLockPassword,
        canResendHostNameTemplate,
        resendHostNameTemplate
      ),
    [
      canResendProfiles,
      resendRequest,
      onProfileResent,
      resendCertificateRequest,
      canRotateRecoveryLockPassword,
      rotateRecoveryLockPassword,
      canResendHostNameTemplate,
      resendHostNameTemplate,
    ]
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

  return (
    <div className={classnames(baseClass, className)}>
      <TableContainer
        columnConfigs={tableConfig}
        data={controls}
        isLoading={false}
        resultsTitle="controls"
        defaultSortHeader="status"
        defaultSortDirection="asc"
        renderCount={() => (
          <TableCount name="controls" count={controls.length} />
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
        onClickRow={(row: Row<IHostMdmProfileWithAddedStatus>) =>
          setSelectedControl(row.original)
        }
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
    </div>
  );
};

export default Controls;
