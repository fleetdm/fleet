/**
 * HostInstallerActionCell is only used on Host details > Software Library card.
 * Displays install/uninstall buttons for host installers.
 * HostInstallerActionButton is reused for install/uninstall buttons
 * in Fleet Desktop > Self-service but with different disabled states and tooltips.
 */

import React, { useState, useEffect, ReactNode } from "react";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import ActionsDropdown from "components/ActionsDropdown";
import { IDropdownOption } from "interfaces/dropdownOption";

import {
  IHostSoftwarePackage,
  IHostAppStoreApp,
  EnhancedSoftwareInstallUninstallStatus,
  IHostSoftwareWithUiStatus,
  SCRIPT_PACKAGE_SOURCES,
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
  moreDisabled?: boolean;
}

export interface IGetActionButtonStateProps {
  hostScriptsEnabled: boolean;
  softwareId: number;
  status: EnhancedSoftwareInstallUninstallStatus | null;
  softwarePackage: IHostSoftwarePackage | null;
  appStoreApp: IHostAppStoreApp | null;
  hostMDMEnrolled?: boolean;
  isMyDevicePage?: boolean;
  installedVersionsDetected: boolean | null;
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
  onClickInstallAction: (
    softwareId: number,
    isSoftwarePackage?: boolean
  ) => void;
  onClickUninstallAction: () => void;
  onClickOpenInstructionsAction?: () => void;
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
  installedVersionsDetected,
}: IGetActionButtonStateProps): IActionButtonState => {
  const isPending = ["pending_install", "pending_uninstall"].includes(
    status || ""
  );

  if (isPending) {
    return {
      installDisabled: true,
      uninstallDisabled: true,
      moreDisabled: true,
    };
  }

  /** My Device page doesn’t enforce tooltips/extra restrictions
   * as these are enforced by the UI to hide when host scripts are
   * not enabled, not enrolled into MDM, etc */
  if (isMyDevicePage) {
    return {
      installDisabled: false,
      uninstallDisabled: false,
      moreDisabled: false,
    };
  }

  /** Host details > Software > Library page has additional tooltips and
   * disabled states than My Device > Self-service page */

  /** If scripts are not enabled, software actions disabled with tooltip on
   * Host details > Software > Library but doesn't show to Fleet Desktop > Self-service. */
  if (!hostScriptsEnabled && !appStoreApp) {
    return {
      installDisabled: true,
      uninstallDisabled: true,
      installTooltip: "To install, turn on host scripts.",
      uninstallTooltip: "To uninstall, turn on host scripts.",
      moreDisabled: !installedVersionsDetected, // Can still reach how to open if scripts are disabled but app is installed
    };
  }

  /** If MDM is not enrolled, software actions disabled with tooltip on
    Host details > Software > Library but doesn't show Fleet Desktop > Self-service */
  if (appStoreApp) {
    return {
      installDisabled: !hostMDMEnrolled,
      uninstallDisabled: true,
      installTooltip: !hostMDMEnrolled
        ? "To install, turn on MDM for this host."
        : undefined,
    };
  }

  return { installDisabled: false, uninstallDisabled: false };
};

const getMoreActionsDropdownOptions = (
  canViewOpenInstructions: boolean,
  canUninstallSoftware: boolean,
  isUninstallDisabled: boolean,
  uninstallTooltip: ReactNode,
  uninstallText: string
) => {
  const options: IDropdownOption[] = [];

  if (canViewOpenInstructions) {
    options.unshift({
      label: "How to open",
      value: "instructions",
      disabled: false,
    });
  }

  if (canUninstallSoftware) {
    options.unshift({
      label: uninstallText,
      value: "uninstall",
      disabled: isUninstallDisabled,
      tooltipContent: uninstallTooltip,
    });
  }

  return options;
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
        variant="inverse"
        type="button"
        className={`${baseClass}__item-action-button`}
        onClick={onClick}
        disabled={disabled}
        size="small"
      >
        <Icon name={icon} color="ui-fleet-black-75" size="small" />
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
  onClickOpenInstructionsAction,
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

  const installedVersionsDetected =
    installed_versions && installed_versions.length > 0;

  const installedTgzPackageDetected =
    software.source === "tgz_packages" &&
    [
      "installed",
      "pending_uninstall",
      "uninstalling",
      "failed_uninstall",
    ].includes(ui_status);

  const isIpaPackage =
    (software.source === "ios_apps" || software.source === "ipados_apps") &&
    !!software_package;

  const canUninstallSoftware =
    !app_store_app &&
    !isIpaPackage &&
    !!software_package &&
    (installedVersionsDetected || installedTgzPackageDetected);

  // Instructions to open software available for macOS apps and Windows programs only
  const canViewOpenInstructions =
    ["apps", "programs"].includes(software.source) &&
    !!installedVersionsDetected;

  const {
    installDisabled,
    installTooltip,
    uninstallDisabled,
    uninstallTooltip,
    moreDisabled,
  } = getActionButtonState({
    hostScriptsEnabled: hostScriptsEnabled || false,
    softwareId: id,
    status,
    appStoreApp: app_store_app,
    hostMDMEnrolled,
    softwarePackage: software_package,
    isMyDevicePage,
    installedVersionsDetected,
  });

  const onSelectOption = (option: string) => {
    switch (option) {
      case "uninstall":
        onClickUninstallAction();
        break;
      case "instructions":
        onClickOpenInstructionsAction && onClickOpenInstructionsAction();
        break;
      default:
    }
  };

  const renderInstallButton = () => (
    <HostInstallerActionButton
      baseClass={baseClass}
      tooltip={installTooltip}
      disabled={installDisabled}
      onClick={() =>
        onClickInstallAction(
          id,
          SCRIPT_PACKAGE_SOURCES.includes(software.source)
        )
      }
      icon={buttonDisplayConfig.install.icon}
      text={buttonDisplayConfig.install.text}
      testId={`${baseClass}__install-button--test`}
    />
  );

  const renderUninstallButton = () => {
    if (!canUninstallSoftware) {
      return null;
    }

    return (
      <HostInstallerActionButton
        baseClass={baseClass}
        tooltip={uninstallTooltip}
        disabled={uninstallDisabled}
        onClick={onClickUninstallAction}
        icon={buttonDisplayConfig.uninstall.icon}
        text={buttonDisplayConfig.uninstall.text}
        testId={`${baseClass}__uninstall-button--test`}
      />
    );
  };

  const renderSecondaryActions = () => {
    // Case: both uninstall + instructions → "More" dropdown
    if (canUninstallSoftware && canViewOpenInstructions) {
      return (
        <div className={`${baseClass}__more-actions-wrapper`}>
          <ActionsDropdown
            className={`${baseClass}__more-actions-dropdown`}
            onChange={onSelectOption}
            placeholder="More"
            options={getMoreActionsDropdownOptions(
              canViewOpenInstructions,
              canUninstallSoftware,
              uninstallDisabled,
              uninstallTooltip,
              buttonDisplayConfig.uninstall.text
            )}
            variant="small-button"
            disabled={moreDisabled}
          />
        </div>
      );
    }

    // Case: uninstall only → Uninstall. button
    if (canUninstallSoftware) {
      return renderUninstallButton();
    }

    // Case: instructions only → How to open button
    if (canViewOpenInstructions && onClickOpenInstructionsAction) {
      return (
        <HostInstallerActionButton
          baseClass={baseClass}
          disabled={false}
          onClick={onClickOpenInstructionsAction}
          icon="info-outline"
          text="How to open"
          testId={`${baseClass}__instructions-button--test`}
        />
      );
    }

    // Case: no secondary actions
    return null;
  };

  if (isMyDevicePage) {
    return (
      <div className={`${baseClass}__item-actions`}>
        {renderInstallButton()}
        {renderSecondaryActions()}
      </div>
    );
  }

  return (
    <div className={`${baseClass}__item-actions`}>
      {renderInstallButton()}
      {renderUninstallButton()}
    </div>
  );
};

export default HostInstallerActionCell;
