import React, { useContext } from "react";

import { MdmEnrollmentStatus } from "interfaces/mdm";
import { AppContext } from "context/app";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { generateHostActionOptions } from "./helpers";

const baseClass = "host-actions-dropdown";

interface IHostActionsDropdownProps {
  hostStatus: string;
  hostMdmEnrollemntStatus: MdmEnrollmentStatus | null;
  doesStoreEncryptionKey?: boolean;
  mdmName?: string;
  onSelect: (value: string) => void;
}

const HostActionsDropdown = ({
  onSelect,
  hostStatus,
  hostMdmEnrollemntStatus,
  doesStoreEncryptionKey,
  mdmName,
}: IHostActionsDropdownProps) => {
  const {
    isPremiumTier = false,
    isGlobalAdmin = false,
    isGlobalMaintainer = false,
    isMdmEnabledAndConfigured = false,
    isTeamAdmin = false,
    isTeamMaintainer = false,
    isSandboxMode = false,
  } = useContext(AppContext);

  const options = generateHostActionOptions({
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isHostOnline: hostStatus === "online",
    isEnrolledInMdm: ["On (automatic)", "On (manual)"].includes(
      hostMdmEnrollemntStatus ?? ""
    ),
    isFleetMdm: mdmName === "Fleet",
    isMdmEnabledAndConfigured,
    doesStoreEncryptionKey: doesStoreEncryptionKey ?? false,
    isSandboxMode,
  });

  // No options to render. Exit early
  if (options.length === 0) return null;

  return (
    <div className={baseClass}>
      <Dropdown
        className={`${baseClass}__host-actions-dropdown`}
        onChange={onSelect}
        placeholder={"Actions"}
        searchable={false}
        options={options}
      />
    </div>
  );
};

export default HostActionsDropdown;
