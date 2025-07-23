/**
 * HostInstallerActionCell is only used on Host details > Software Library card.
 * Displays install/uninstall buttons for host installers.
 * HostInstallerActionButton is reused for install/uninstall buttons
 * in Fleet Desktop > Self-service but with different disabled states and tooltips.
 */

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

export interface IGetActionButtonStateProps {
  hostScriptsEnabled: boolean;
  softwareId: number;
  status: SoftwareInstallStatus | null;
  softwarePackage: IHostSoftwarePackage | null;
  appStoreApp: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
  isMyDevicePage?: boolean;
}

interface IHostInstallerActionButtonProps {
  baseClass: string;
  tooltip?: React.ReactNode;
  disabled: boolean;
  onClick: () => void;
  icon: IconNames;
  text: string;
  testId?: string;
}

interface IHostInstallerActionCellProps {
  software: IHostSoftwareWithUiStatus;
  onClickInstallAction: (softwareId: number) => void;
  onClickUninstallAction: () => void;
  baseClass: string;
  hostScriptsEnabled?: boolean;
  hostMDMEnrolled?: boolean;
  /** Different disabled states and tooltips compared to host details > library and fleet desktop > self-service */
  isMyDevicePage?: boolean;
}

export const getActionButtonState = ({
  hostScriptsEnabled,
  status,
  appStoreApp,
  hostMDMEnrolled,
  isMyDevicePage,
}: IGetActionButtonStateProps): IActionButtonState => {
  const pendingStatuses = ["pending_install", "pending_uninstall"];
  let installDisabled = false;
  let uninstallDisabled = false;
  let installTooltip: React.ReactNode | undefined;
  let uninstallTooltip: React.ReactNode | undefined;

  // Action buttons are always disabled if status is pending for both
  // Host details > Software > library page and  My Device > Self-service page
  if (pendingStatuses.includes(status || "")) {
    installDisabled = true;
    uninstallDisabled = true;
  }

  /** Host details > Software > Library page has additional tooltips and disabled states
   * than My Device > Self-service page for disabled host scripts and mdm unenrolled
   *
   * If scripts are not enabled, software actions disabled with tooltip on
   * Host details > Software > Library but doesn't show to Fleet Desktop > Self-service.
   *
   * If MDM is not enrolled, software actions disabled with tooltip on
   * Host details > Software > Library but doesn't show Fleet Desktop > Self-service */
  if (!isMyDevicePage) {
    if (!hostScriptsEnabled && !appStoreApp) {
      installDisabled = true;
      uninstallDisabled = true;
      installTooltip = "To install, turn on host scripts.";
      uninstallTooltip = "To uninstall, turn on host scripts.";
    }

    if (appStoreApp) {
      uninstallDisabled = true;
      if (!hostMDMEnrolled) {
        installDisabled = true;
        installTooltip = "To install, turn on MDM for this host.";
      }
    }
  }

  return {
    installDisabled,
    installTooltip: installTooltip || undefined,
    uninstallDisabled,
    uninstallTooltip: uninstallTooltip || undefined,
  };
};

export const HostInstallerActionButton = ({
  baseClass,
  tooltip,
  disabled,
  onClick,
  icon,
  text,
  testId,
}: IHostInstallerActionButtonProps) => (
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

/** HostInstallerActionCell component has different disabled states
 * and tooltips for Host details > Library HostInstallerActionCell than
 * Fleet Desktop > Self-service HostInstallerActionCell */
export const HostInstallerActionCell = ({
  software,
  onClickInstallAction,
  onClickUninstallAction,
  baseClass,
  hostScriptsEnabled,
  hostMDMEnrolled,
  isMyDevicePage = false,
}: IHostInstallerActionCellProps) => {
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
    isMyDevicePage,
  });

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

  const canUninstallSoftware =
    !app_store_app &&
    !!software_package &&
    installed_versions &&
    installed_versions.length > 0;

  return (
    <div className={`${baseClass}__item-actions`}>
      <HostInstallerActionButton
        baseClass={baseClass}
        tooltip={installTooltip}
        disabled={installDisabled}
        onClick={() => onClickInstallAction(id)}
        icon={buttonDisplayConfig.install.icon}
        text={buttonDisplayConfig.install.text}
        testId={`${baseClass}__install-button--test`}
      />
      {canUninstallSoftware && (
        <HostInstallerActionButton
          baseClass={baseClass}
          tooltip={uninstallTooltip}
          disabled={uninstallDisabled}
          onClick={onClickUninstallAction}
          icon={buttonDisplayConfig.uninstall.icon}
          text={buttonDisplayConfig.uninstall.text}
          testId={`${baseClass}__uninstall-button--test`}
        />
      )}
    </div>
  );
};

export default HostInstallerActionCell;
