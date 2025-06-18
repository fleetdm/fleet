import React, { useState, useEffect } from "react";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import {
  IHostSoftware,
  IHostSoftwarePackage,
  IHostAppStoreApp,
  SoftwareInstallStatus,
} from "interfaces/software";
import {
  DisplayActionItems,
  getInstallButtonText,
  getInstallButtonIcon,
  getUninstallButtonText,
  getUninstallButtonIcon,
} from "../../Software/SelfService/SelfServiceTableConfig";

export interface generateActionsProps {
  hostScriptsEnabled: boolean;
  softwareId: number;
  status: SoftwareInstallStatus | null;
  software_package: IHostSoftwarePackage | null;
  app_store_app: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
}

interface IInstallerActionCellProps {
  software: IHostSoftware;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  baseClass: string;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
}

interface IButtonActionState {
  installDisabled: boolean;
  installTooltip?: React.ReactNode;
  uninstallDisabled: boolean;
  uninstallTooltip?: React.ReactNode;
}

export const getButtonActionState = ({
  hostScriptsEnabled,
  status,
  app_store_app,
  hostMDMEnrolled,
}: generateActionsProps): IButtonActionState => {
  const pendingStatuses = ["pending_install", "pending_uninstall"];
  let installDisabled = false;
  let uninstallDisabled = false;
  let installTooltip: React.ReactNode | undefined;
  let uninstallTooltip: React.ReactNode | undefined;

  if (!hostScriptsEnabled && !app_store_app) {
    installDisabled = true;
    uninstallDisabled = true;
    installTooltip = "To install, turn on host scripts.";
    uninstallTooltip = "To uninstall, turn on host scripts.";
  }

  if (pendingStatuses.includes(status || "")) {
    installDisabled = true;
    uninstallDisabled = true;
  }

  if (app_store_app) {
    uninstallDisabled = true;
    if (!hostMDMEnrolled) {
      installDisabled = true;
      installTooltip = "To install, turn on MDM for this host.";
    }
  }

  return {
    installDisabled,
    installTooltip: installTooltip || undefined,
    uninstallDisabled,
    uninstallTooltip: uninstallTooltip || undefined,
  };
};

export const InstallerActionCell = ({
  software,
  onClickInstallAction,
  onClickUninstallAction,
  baseClass,
  hostScriptsEnabled,
  hostMDMEnrolled,
}: IInstallerActionCellProps) => {
  const { id, status, software_package, app_store_app } = software;
  const {
    installDisabled,
    installTooltip,
    uninstallDisabled,
    uninstallTooltip,
  } = getButtonActionState({
    hostScriptsEnabled: hostScriptsEnabled || false,
    softwareId: id,
    status,
    app_store_app,
    hostMDMEnrolled,
    software_package,
  });

  const [
    displayActionItems,
    setDisplayActionItems,
  ] = useState<DisplayActionItems>({
    install: {
      text: getInstallButtonText(status),
      icon: getInstallButtonIcon(status),
    },
    uninstall: {
      text: getUninstallButtonText(status),
      icon: getUninstallButtonIcon(status),
    },
  });

  useEffect(() => {
    if (status !== "pending_install" && status !== "pending_uninstall") {
      setDisplayActionItems({
        install: {
          text: getInstallButtonText(status),
          icon: getInstallButtonIcon(status),
        },
        uninstall: {
          text: getUninstallButtonText(status),
          icon: getUninstallButtonIcon(status),
        },
      });
    }
  }, [status]);

  return (
    <div className={`${baseClass}__item-actions`}>
      <div className={`${baseClass}__item-action`}>
        <TooltipWrapper
          tipContent={installTooltip}
          underline={false}
          showArrow
          position="top"
        >
          <Button
            variant="text-icon"
            type="button"
            className={`${baseClass}__item-action-button`}
            onClick={() => onClickInstallAction(id)}
            disabled={installDisabled}
          >
            <Icon
              name={displayActionItems.install.icon}
              color="core-fleet-blue"
              size="small"
            />
            <span data-testid={`${baseClass}__install-button--test`}>
              {displayActionItems.install.text}
            </span>
          </Button>
        </TooltipWrapper>
      </div>
      <div className={`${baseClass}__item-action`}>
        {app_store_app
          ? null
          : software_package && (
              <TooltipWrapper
                tipContent={uninstallTooltip}
                underline={false}
                showArrow
                position="top"
              >
                <Button
                  variant="text-icon"
                  type="button"
                  className={`${baseClass}__item-action-button`}
                  onClick={() => onClickUninstallAction(id)}
                  disabled={uninstallDisabled}
                >
                  <Icon
                    name={displayActionItems.uninstall.icon}
                    color="core-fleet-blue"
                    size="small"
                  />
                  <span data-testid={`${baseClass}__uninstall-button--test`}>
                    {displayActionItems.uninstall.text}
                  </span>
                </Button>
              </TooltipWrapper>
            )}
      </div>
    </div>
  );
};

export default InstallerActionCell;
