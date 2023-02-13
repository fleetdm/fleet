import React, { useContext } from "react";

import { MdmEnrollmentStatus } from "interfaces/mdm";
import permissionUtils from "utilities/permissions";
import { AppContext } from "context/app";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { generateHostActionOptions } from "./helpers";

const baseClass = "host-actions-dropdown";

interface IHostActionsDropdownProps {
  onSelect: (value: string) => void;
  teamId: number | null;
  hostStatus: string;
  hostMdmEnrollemntStatus: MdmEnrollmentStatus | null;
  doesStoreEncryptionKey?: boolean;
}

const HostActionsDropdown = ({
  onSelect,
  teamId,
  hostStatus,
  hostMdmEnrollemntStatus,
  doesStoreEncryptionKey,
}: IHostActionsDropdownProps) => {
  const {
    currentUser,
    isPremiumTier = false,
    isGlobalAdmin = false,
    isGlobalMaintainer = false,
  } = useContext(AppContext);

  const options = generateHostActionOptions({
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin: permissionUtils.isTeamAdmin(currentUser, teamId ?? null),
    isTeamMaintainer: permissionUtils.isTeamMaintainer(
      currentUser,
      teamId ?? null
    ),
    isHostOnline: hostStatus === "online",
    isEnrolledInMdm: ["On (automatic)", "On (manual)"].includes(
      hostMdmEnrollemntStatus ?? ""
    ),
    doesStoreEncryptionKey: doesStoreEncryptionKey ?? false,
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
