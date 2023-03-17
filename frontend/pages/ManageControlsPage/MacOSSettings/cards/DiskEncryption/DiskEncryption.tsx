import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ITeamConfig } from "interfaces/team";
import mdmAPI from "services/entities/mdm";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import DiskEncryptionTable from "./components/DiskEncryptionTable";

const baseClass = "disk-encryption";
interface IDiskEncryptionProps {
  currentTeamId?: number;
}

const DiskEncryption = ({ currentTeamId }: IDiskEncryptionProps) => {
  const { isPremiumTier, config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const defaultShowDiskEncryption = currentTeamId
    ? false
    : config?.mdm.macos_settings.enable_disk_encryption ?? false;

  const [showAggregate, setShowAggregate] = useState(defaultShowDiskEncryption);
  const [diskEncryptionEnabled, setDiskEncryptionEnabled] = useState(
    defaultShowDiskEncryption
  );

  const onToggleCheckbox = (value: boolean) => {
    setDiskEncryptionEnabled(value);
  };

  useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId ?? 0),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: Boolean(currentTeamId),
      select: (res) => res.team,
      onSuccess: (res) => {
        const enableDiskEncryption =
          res.mdm?.macos_settings.enable_disk_encryption ?? false;
        setDiskEncryptionEnabled(enableDiskEncryption);
        setShowAggregate(enableDiskEncryption);
      },
    }
  );

  const onUpdateDiskEncryption = async () => {
    try {
      await mdmAPI.updateAppleMdmSettings(diskEncryptionEnabled, currentTeamId);
      renderFlash(
        "success",
        "Successfully updated disk encryption enforcement!"
      );
      setShowAggregate(diskEncryptionEnabled);
    } catch {
      console.error("error updating");
      renderFlash(
        "error",
        "Could not update the disk encryption enforcement. Please try again."
      );
    }
  };

  return (
    <div className={baseClass}>
      <h2>Disk encryption</h2>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <>
          {/* remove && false to show the table once the API is finished */}
          {showAggregate && false ? (
            <DiskEncryptionTable currentTeamId={currentTeamId} />
          ) : null}
          <Checkbox
            onChange={onToggleCheckbox}
            value={diskEncryptionEnabled}
            className={`${baseClass}__checkbox`}
          >
            On
          </Checkbox>
          <p>
            Apple calls this “FileVault.” If turned on, hosts&apos; disk
            encryption keys will be stored in Fleet.{" "}
            <CustomLink
              text="Learn more"
              url="https://fleetdm.com/docs/using-fleet/mobile-device-management#disk-encryption"
              newTab
            />
          </p>
          <Button
            className={`${baseClass}__save-button`}
            onClick={onUpdateDiskEncryption}
          >
            Save
          </Button>
        </>
      )}
    </div>
  );
};

export default DiskEncryption;
