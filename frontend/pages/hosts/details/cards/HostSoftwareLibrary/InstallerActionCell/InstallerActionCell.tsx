import React, { useState, useEffect } from "react";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import {
  IHostSoftwarePackage,
  IHostAppStoreApp,
  SoftwareInstallStatus,
  IHostSoftwareWithUiStatus,
} from "interfaces/software";
import { IconNames } from "components/icons";
import {
  getInstallerActionButtonConfig,
  IButtonDisplayConfig,
} from "../../Software/helpers";

interface IActionButtonState {
  installDisabled: boolean;
  installTooltip?: React.ReactNode;
  uninstallDisabled: boolean;
  uninstallTooltip?: React.ReactNode;
}

export interface IActionButtonProps {
  hostScriptsEnabled: boolean;
  softwareId: number;
  status: SoftwareInstallStatus | null;
  softwarePackage: IHostSoftwarePackage | null;
  appStoreApp: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
}

interface IInstallerActionButtonProps {
  baseClass: string;
  tooltip?: React.ReactNode;
  disabled: boolean;
  onClick: () => void;
  icon: IconNames;
  text: string;
  testId?: string;
}

interface IInstallerActionCellProps {
  software: IHostSoftwareWithUiStatus;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: (softwareId: number) => void;
  baseClass: string;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
}

export const getActionButtonState = ({
  hostScriptsEnabled,
  status,
  appStoreApp,
  hostMDMEnrolled,
}: IActionButtonProps): IActionButtonState => {
  const pendingStatuses = ["pending_install", "pending_uninstall"];
  let installDisabled = false;
  let uninstallDisabled = false;
  let installTooltip: React.ReactNode | undefined;
  let uninstallTooltip: React.ReactNode | undefined;

  if (!hostScriptsEnabled && !appStoreApp) {
    installDisabled = true;
    uninstallDisabled = true;
    installTooltip = "To install, turn on host scripts.";
    uninstallTooltip = "To uninstall, turn on host scripts.";
  }

  if (pendingStatuses.includes(status || "")) {
    installDisabled = true;
    uninstallDisabled = true;
  }

  if (appStoreApp) {
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

export const InstallerActionButton = ({
  baseClass,
  tooltip,
  disabled,
  onClick,
  icon,
  text,
  testId,
}: IInstallerActionButtonProps) => (
  <div className={`${baseClass}__item-action`}>
    <TooltipWrapper
      tipContent={tooltip}
      underline={false}
      showArrow
      position="top"
    >
      <Button
        variant="text-icon"
        type="button"
        className={`${baseClass}__item-action-button`}
        onClick={onClick}
        disabled={disabled}
      >
        <Icon name={icon} color="core-fleet-blue" size="small" />
        <span data-testid={testId}>{text}</span>
      </Button>
    </TooltipWrapper>
  </div>
);

export const InstallerActionCell = ({
  software,
  onClickInstallAction,
  onClickUninstallAction,
  baseClass,
  hostScriptsEnabled,
  hostMDMEnrolled,
}: IInstallerActionCellProps) => {
  const {
    id,
    status,
    software_package,
    app_store_app,
    ui_status,
    installed_versions,
  } = software;
  const {
    installDisabled,
    installTooltip,
    uninstallDisabled,
    uninstallTooltip,
  } = getActionButtonState({
    hostScriptsEnabled: hostScriptsEnabled || false,
    softwareId: id,
    status,
    appStoreApp: app_store_app,
    hostMDMEnrolled,
    softwarePackage: software_package,
  });

  const canUninstallSoftware =
    !app_store_app && installed_versions && installed_versions.length > 0;

  // buttonDisplayConfig is used to track the display text and icons of the install and uninstall button
  const [
    buttonDisplayConfig,
    setButtonDisplayConfig,
  ] = useState<IButtonDisplayConfig>({
    install: getInstallerActionButtonConfig("install", ui_status),
    uninstall: getInstallerActionButtonConfig("uninstall", ui_status),
  });

  useEffect(() => {
    // We update the text/icon only when we see a change to a non-pending status
    // Pending statuses keep the original text shown (e.g. "Retry" text on failed
    // install shouldn't change to "Install" text because it was clicked and went
    // pending. Once the status is no longer pending, like 'installed' the
    // text will update to "Reinstall")
    if (status !== "pending_install" && status !== "pending_uninstall") {
      setButtonDisplayConfig({
        install: getInstallerActionButtonConfig("install", ui_status),
        uninstall: getInstallerActionButtonConfig("uninstall", ui_status),
      });
    }
  }, [status, ui_status]);

  return (
    <div className={`${baseClass}__item-actions`}>
      <InstallerActionButton
        baseClass={baseClass}
        tooltip={installTooltip}
        disabled={installDisabled}
        onClick={() => onClickInstallAction(id)}
        icon={buttonDisplayConfig.install.icon}
        text={buttonDisplayConfig.install.text}
        testId={`${baseClass}__install-button--test`}
      />
      {canUninstallSoftware && software_package && (
        <InstallerActionButton
          baseClass={baseClass}
          tooltip={uninstallTooltip}
          disabled={uninstallDisabled}
          onClick={() => onClickUninstallAction(id)}
          icon={buttonDisplayConfig.uninstall.icon}
          text={buttonDisplayConfig.uninstall.text}
          testId={`${baseClass}__uninstall-button--test`}
        />
      )}
    </div>
  );
};

export default InstallerActionCell;
