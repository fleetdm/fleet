import React, { useContext } from "react";

import { MdmEnrollmentStatus } from "interfaces/mdm";
import permissions from "utilities/permissions";
import { AppContext } from "context/app";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
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
  mdmName?: string;
  hostPlatform?: string;
  onSelect: (value: string) => void;
}

const HostActionsDropdown = ({
  hostTeamId,
  hostStatus,
  hostMdmEnrollmentStatus,
  hostMdmDeviceStatus,
  doesStoreEncryptionKey,
  mdmName,
  hostPlatform = "",
  onSelect,
}: IHostActionsDropdownProps) => {
  const {
    isPremiumTier = false,
    isGlobalAdmin = false,
    isGlobalMaintainer = false,
    isMdmEnabledAndConfigured = false,
    isSandboxMode = false,
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
    isFleetMdm: mdmName === "Fleet",
    isMdmEnabledAndConfigured,
    doesStoreEncryptionKey: doesStoreEncryptionKey ?? false,
    isSandboxMode,
    hostMdmDeviceStatus,
  });

  // No options to render. Exit early
  if (options.length === 0) return null;

  return (
    <div className={baseClass}>
      <Dropdown
        className={`${baseClass}__host-actions-dropdown`}
        onChange={onSelect}
        placeholder="Actions"
        searchable={false}
        options={options}
      />
    </div>
  );
};

export default HostActionsDropdown;
