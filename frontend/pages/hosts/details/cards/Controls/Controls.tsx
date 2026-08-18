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
  canResendProfiles: boolean;
  canRotateRecoveryLockPassword?: boolean;
  canResendHostNameTemplate?: boolean;
  /** Whether the user can reach the Controls page the empty state links to. */
  canAddControls?: boolean;
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
  canResendProfiles,
  canRotateRecoveryLockPassword = false,
  canResendHostNameTemplate = false,
  canAddControls = false,
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

  const onAddControls = useCallback(() => {
    router.push(getPathWithQueryParams(PATHS.CONTROLS, { fleet_id: teamId }));
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
            info={`No controls have been added for ${
              isDeviceUser ? "your device" : "this host"
            }.`}
            primaryButton={
              !isDeviceUser && canAddControls ? (
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
