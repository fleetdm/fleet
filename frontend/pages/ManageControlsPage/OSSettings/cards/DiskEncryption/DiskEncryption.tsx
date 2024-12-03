import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { ITeamConfig } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import diskEncryptionAPI from "services/entities/disk_encryption";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import TooltipWrapper from "components/TooltipWrapper";

import DiskEncryptionTable from "./components/DiskEncryptionTable";

const baseClass = "disk-encryption";
interface IDiskEncryptionProps {
  currentTeamId: number;
  onMutation: () => void;
  router: InjectedRouter;
}

const DiskEncryption = ({
  currentTeamId,
  onMutation,
  router,
}: IDiskEncryptionProps) => {
  const { isPremiumTier, config, setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const defaultShowDiskEncryption = currentTeamId
    ? false
    : config?.mdm.enable_disk_encryption ?? false;

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
        const enableDiskEncryption = res.mdm?.enable_disk_encryption ?? false;
        setDiskEncryptionEnabled(enableDiskEncryption);
        setShowAggregate(enableDiskEncryption);
        setIsLoadingTeam(false);
      },
    }
  );

  const onUpdateDiskEncryption = async () => {
    try {
      await diskEncryptionAPI.updateDiskEncryption(
        diskEncryptionEnabled,
        currentTeamId
      );
      renderFlash(
        "success",
        "Successfully updated disk encryption enforcement!"
      );
      onMutation();
      setShowAggregate(diskEncryptionEnabled);
      if (currentTeamId === 0) {
        getUpdatedAppConfig();
      }
    } catch (e) {
      if (getErrorReason(e).includes("Missing required private key")) {
        const link =
          "https://fleetdm.com/learn-more-about/fleet-server-private-key";
        renderFlash(
          "error",
          <>
            Could&apos;t enable disk encryption. Missing required private key.
            Learn how to configure the private key here:{" "}
            <a href={link}>{link}</a>
          </>
        );
      } else {
        renderFlash(
          "error",
          "Could not update the disk encryption enforcement. Please try again."
        );
      }
    }
  };

  if (currentTeamId === 0 && isLoadingTeam) {
    setIsLoadingTeam(false);
  }

  const getTipContent = (platform: "windows" | "macOS" | "linux") => {
    if (platform === "linux") {
      return (
        <>
          For Ubuntu, Kubuntu, and Fedora Linux.
          <br />
          Currently, full disk encryption must be turned on{" "}
          <b>
            during OS
            <br />
            setup
          </b>
          . If disk encryption is off, the end user must re-install
          <br />
          their operating system.
        </>
      );
    }
    const [AppleOrWindows, DEMethod] =
      platform === "windows"
        ? ["Windows", "BitLocker"]
        : ["Apple", "FileVault"];
    return (
      <>
        {AppleOrWindows} MDM must be turned on in{" "}
        <a href="/settings/integrations/mdm">
          <b>Settings</b> &gt; <b>Integrations</b> &gt;{" "}
          <b>Mobile Device Management (MDM)</b>
        </a>{" "}
        to enforce disk encryption via {DEMethod}.
      </>
    );
  };

  const subTitle = (
    <>
      Disk encryption is available on{" "}
      <TooltipWrapper tipContent={getTipContent("macOS")}>macOS</TooltipWrapper>
      ,{" "}
      <TooltipWrapper tipContent={getTipContent("windows")}>
        Windows
      </TooltipWrapper>
      , and{" "}
      <TooltipWrapper tipContent={getTipContent("linux")}>Linux</TooltipWrapper>{" "}
      hosts.
    </>
  );

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Disk encryption"
        subTitle={subTitle}
        alignLeftHeaderVertically
        greySubtitle
      />
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
                <DiskEncryptionTable
                  currentTeamId={currentTeamId}
                  router={router}
                />
              )}
              <Checkbox
                onChange={onToggleCheckbox}
                value={diskEncryptionEnabled}
                className={`${baseClass}__checkbox`}
              >
                Turn on disk encryption
              </Checkbox>
              <p>
                If turned on, hosts&apos; disk encryption keys will be stored in
                Fleet{" "}
                <CustomLink
                  text="Learn more"
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/mdm-disk-encryption`}
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
