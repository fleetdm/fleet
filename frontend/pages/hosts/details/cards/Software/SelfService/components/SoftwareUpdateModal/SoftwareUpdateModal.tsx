import { flatMap } from "lodash";
import React from "react";

import Button from "components/buttons/Button";
import IconStatusMessage from "components/IconStatusMessage";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import {
  IHostSoftware,
  SoftwareInstallUninstallStatus,
} from "interfaces/software";
import { compareVersions } from "pages/hosts/details/cards/Software/helpers";
import InventoryVersions from "pages/hosts/details/components/InventoryVersions";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

const baseClass = "software-update-modal";

const DEFAULT_MACOS_APP_PATH_REGEX = /^\/Applications\/[^/]+\.app$/;

/** Paths of outdated macOS app copies that updating likely won't replace,
 * because the installer writes to a different path. */
const getStalePaths = (
  software: IHostSoftware,
  installerVersion?: string
): string[] => {
  if (software.source !== "apps" || !installerVersion) return [];

  const versions = software.installed_versions || [];
  const isOutdated = (version: string) =>
    compareVersions(version, installerVersion) === -1;
  const outdated = versions.filter((iv) => isOutdated(iv.version));
  const current = versions.filter((iv) => !isOutdated(iv.version));
  const currentPaths = new Set(flatMap(current, (iv) => iv.installed_paths));

  const outdatedPaths = Array.from(
    new Set(flatMap(outdated, (iv) => iv.installed_paths))
  ).filter((path) => !currentPaths.has(path));

  // With no up-to-date copy, a lone outdated copy at the default location is
  // a regular update, so only flag copies elsewhere.
  if (current.length === 0) {
    return outdatedPaths.filter(
      (path) => !DEFAULT_MACOS_APP_PATH_REGEX.test(path)
    );
  }
  return outdatedPaths;
};

const renderPathList = (paths: string[]) =>
  paths.map((path, i) => {
    const isLast = i === paths.length - 1;
    let separator = "";
    if (i > 0 && !isLast) separator = ", ";
    else if (i > 0) separator = paths.length > 2 ? ", and " : " and ";
    return (
      <React.Fragment key={path}>
        {separator}
        <b>{path}</b>
      </React.Fragment>
    );
  });

const StalePathsMessage = ({ paths }: { paths: string[] }) => {
  const isPlural = paths.length > 1;
  return (
    <>
      {isPlural ? "Older versions are" : "An older version is"} installed at{" "}
      {renderPathList(paths)}. Updating may not replace{" "}
      {isPlural ? "them" : "it"}. To clear this update, delete the older{" "}
      {isPlural ? "versions" : "version"}.
    </>
  );
};

// Status message render helper
interface IStatusMessageProps {
  hostDisplayName: string;
  isDeviceUser: boolean;
  softwareStatus: SoftwareInstallUninstallStatus | null;
  softwareName: string;
  installerName: string;
  installerVersion?: string;
  stalePaths: string[];
}
const StatusMessage = ({
  hostDisplayName,
  isDeviceUser,
  softwareStatus,
  softwareName,
  installerName,
  installerVersion,
  stalePaths,
}: IStatusMessageProps) => {
  const renderMessage = () => {
    if (softwareStatus === "pending_install") {
      return (
        <>
          Fleet is updating or will update <b>{softwareName}</b>
          {installerName && ` (${installerName})`} on <b>{hostDisplayName}</b>{" "}
          when it comes online.
        </>
      );
    }
    if (isDeviceUser) {
      return (
        <>
          Update <b>{softwareName}</b> to {installerVersion}.
        </>
      );
    }
    return (
      <>
        New version of <b>{softwareName}</b>
        {installerVersion && ` (${installerVersion})`} is available. Update the
        current version on <b>{hostDisplayName}</b>.
      </>
    );
  };

  return (
    <IconStatusMessage
      className={`${baseClass}__status-message`}
      iconName={
        softwareStatus === "pending_install"
          ? "pending-outline"
          : "error-outline"
      }
      iconColor="ui-fleet-black-50"
      message={
        <span>
          {renderMessage()}
          {softwareStatus !== "pending_install" && stalePaths.length > 0 && (
            <>
              {" "}
              <StalePathsMessage paths={stalePaths} />
            </>
          )}
        </span>
      }
    />
  );
};

interface ISoftwareUpdateModalProps {
  hostDisplayName: string;
  software: IHostSoftware;
  onExit: () => void;
  isDeviceUser?: boolean;
  /** Hides the Update button; modal is view-only */
  disableUpdate?: boolean;
  /** Currently API for updating is the same as installing */
  onUpdate: (id: number) => void;
}

const SoftwareUpdateModal = ({
  hostDisplayName,
  software,
  isDeviceUser = false,
  disableUpdate = false,
  onExit,
  onUpdate,
}: ISoftwareUpdateModalProps) => {
  const {
    id,
    status,
    name,
    display_name,
    installed_versions,
    software_package,
    app_store_app,
  } = software;
  const installerName = software_package?.name || "";
  const installerVersion = software_package?.version || app_store_app?.version;
  const softwareDisplayName = getDisplayedSoftwareName(name, display_name);

  const onClickUpdate = () => {
    onUpdate(id);
    onExit();
  };

  const hasCurrentVersions =
    installed_versions && installed_versions.length > 0;
  const showCurrentVersions =
    hasCurrentVersions && software.status !== "pending_install";

  return (
    <Modal title="Update details" className={baseClass} onExit={onExit}>
      <div className={`${baseClass}__modal-content`}>
        <StatusMessage
          hostDisplayName={hostDisplayName}
          isDeviceUser={isDeviceUser}
          softwareStatus={status}
          softwareName={softwareDisplayName}
          installerName={installerName}
          installerVersion={installerVersion}
          stalePaths={getStalePaths(software, installerVersion)}
        />
        {showCurrentVersions && <InventoryVersions hostSoftware={software} />}
      </div>
      <ModalFooter
        primaryButtons={
          status === "pending_install" || disableUpdate ? (
            <Button type="submit" onClick={onExit}>
              Close
            </Button>
          ) : (
            <>
              <Button variant="secondary" onClick={onExit}>
                Cancel
              </Button>
              <Button type="submit" onClick={onClickUpdate}>
                Update
              </Button>
            </>
          )
        }
      />
    </Modal>
  );
};

export default SoftwareUpdateModal;
