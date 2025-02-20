import React, { useContext } from "react";

import { MdmEnrollmentStatus } from "interfaces/mdm";
import permissions from "utilities/permissions";
import { AppContext } from "context/app";

import ActionsDropdown from "components/ActionsDropdown";
import { generateHostActionOptions } from "./helpers";
import { HostMdmDeviceStatusUIState } from "../../helpers";

const baseClass = "host-actions-dropdown";

interface IHostActionsDropdownProps {
  hostTeamId: number | null;
  hostStatus: string;
  hostMdmEnrollmentStatus: MdmEnrollmentStatus | null;
  /** This represents the mdm managed host device status (e.g. unlocked, locked,
   * unlocking, locking, ...etc) */
  hostMdmDeviceStatus: HostMdmDeviceStatusUIState;
  doesStoreEncryptionKey?: boolean;
  isConnectedToFleetMdm?: boolean;
  hostPlatform?: string;
  onSelect: (value: string) => void;
  hostScriptsEnabled: boolean | null;
}

const HostActionsDropdown = ({
  hostTeamId,
  hostStatus,
  hostMdmEnrollmentStatus,
  hostMdmDeviceStatus,
  doesStoreEncryptionKey,
  isConnectedToFleetMdm,
  hostPlatform = "",
  hostScriptsEnabled = false,
  onSelect,
}: IHostActionsDropdownProps) => {
  const {
    isPremiumTier = false,
    isGlobalAdmin = false,
    isGlobalMaintainer = false,
    isMacMdmEnabledAndConfigured = false,
    isWindowsMdmEnabledAndConfigured = false,
    currentUser,
  } = useContext(AppContext);

  if (!currentUser) return null;

  const isTeamAdmin = permissions.isTeamAdmin(currentUser, hostTeamId);
  const isTeamMaintainer = permissions.isTeamMaintainer(
    currentUser,
    hostTeamId
  );
  const isTeamObserver = permissions.isTeamObserver(currentUser, hostTeamId);
  const isGlobalObserver = permissions.isGlobalObserver(currentUser);

  const options = generateHostActionOptions({
    hostPlatform,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isGlobalObserver,
    isTeamAdmin,
    isTeamMaintainer,
    isTeamObserver,
    isHostOnline: hostStatus === "online",
    isEnrolledInMdm: ["On (automatic)", "On (manual)"].includes(
      hostMdmEnrollmentStatus ?? ""
    ),
    isConnectedToFleetMdm,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    doesStoreEncryptionKey: doesStoreEncryptionKey ?? false,
    hostMdmDeviceStatus,
    hostScriptsEnabled,
  });

  // No options to render. Exit early
  if (options.length === 0) return null;

  return (
    <div className={baseClass}>
      <ActionsDropdown
        className={`${baseClass}__host-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        options={options}
        menuAlign="right"
      />
    </div>
  );
};

export default HostActionsDropdown;
