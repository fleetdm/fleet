import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ITeamConfig } from "interfaces/team";
import mdmAPI from "services/entities/mdm";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";

import DiskEncryptionTable from "./components/DiskEncryptionTable";

const baseClass = "disk-encryption";
interface IDiskEncryptionProps {
  currentTeamId: number;
  onMutation: () => void;
}

const DiskEncryption = ({
  currentTeamId,
  onMutation,
}: IDiskEncryptionProps) => {
  const { isPremiumTier, config, setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const defaultShowDiskEncryption = currentTeamId
    ? false
    : config?.mdm.macos_settings.enable_disk_encryption ?? false;

  const [isLoadingTeam, setIsLoadingTeam] = useState(true);

  const [showAggregate, setShowAggregate] = useState(defaultShowDiskEncryption);
  const [diskEncryptionEnabled, setDiskEncryptionEnabled] = useState(
    defaultShowDiskEncryption
  );

  // because we pull the default state for no teams from the config,
  // we need to update the config when the user toggles the checkbox
  const getUpdatedAppConfig = async () => {
    try {
      const updatedConfig = await configAPI.loadAll();
      setConfig(updatedConfig);
    } catch {
      renderFlash(
        "error",
        "Could not retrieve updated app config. Please try again."
      );
    }
  };

  const onToggleCheckbox = (value: boolean) => {
    setDiskEncryptionEnabled(value);
  };

  useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
      enabled: currentTeamId !== 0,
      select: (res) => res.team,
      onSuccess: (res) => {
        const enableDiskEncryption =
          res.mdm?.macos_settings.enable_disk_encryption ?? false;
        setDiskEncryptionEnabled(enableDiskEncryption);
        setShowAggregate(enableDiskEncryption);
        setIsLoadingTeam(false);
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
      onMutation();
      setShowAggregate(diskEncryptionEnabled);
      if (currentTeamId === 0) {
        getUpdatedAppConfig();
      }
    } catch {
      renderFlash(
        "error",
        "Could not update the disk encryption enforcement. Please try again."
      );
    }
  };

  if (currentTeamId === 0 && isLoadingTeam) {
    setIsLoadingTeam(false);
  }

  return (
    <div className={baseClass}>
      <h2>Disk encryption</h2>
      {!isPremiumTier ? (
        <PremiumFeatureMessage
          className={`${baseClass}__premium-feature-message`}
        />
      ) : (
        <>
          {isLoadingTeam ? (
            <Spinner />
          ) : (
            <div className="disk-encryption-content">
              {showAggregate && (
                <DiskEncryptionTable currentTeamId={currentTeamId} />
              )}
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
                  url="https://fleetdm.com/docs/using-fleet/mdm-disk-encryption"
                  newTab
                />
              </p>
              <Button
                className={`${baseClass}__save-button`}
                onClick={onUpdateDiskEncryption}
              >
                Save
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default DiskEncryption;
