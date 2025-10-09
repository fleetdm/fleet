/** This component is intentionally separate from SoftwareInstallDetailsModal
 * because it handles payload-free/script-based package installs (e.g. sh_packages or ps1_packages)
 *
 * Key differences from SoftwareInstallDetailsModal:
 * - Uses Script/Run/Rerun language in UI instead of Install/Retry.
 * - Omits current versions section (no InventoryVersions display).
 * - Omits post-install script output.
 *
 * Keeping these components and its tests separate improves maintainability and clarity
 */

import React, { useState } from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";
import { AxiosError } from "axios";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import {
  IHostSoftware,
  ISoftwareScriptResult,
  ISoftwareInstallResults,
} from "interfaces/software";
import softwareAPI from "services/entities/software";
import deviceUserAPI from "services/entities/device_user";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner/Spinner";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";

import {
  SCRIPT_DETAILS_STATUS_ICONS,
  getScriptDetailsStatusPredicate,
} from "../constants";

const baseClass = "software-script-details-modal";

export type IPackageInstallDetails = {
  host_display_name?: string;
  install_uuid?: string; // not actually optional
};

export const renderContactOption = (url?: string) => (
  <>
    {" "}
    or{" "}
    {url ? (
      <CustomLink url={url} text="contact your IT admin" newTab />
    ) : (
      "contact your IT admin"
    )}
  </>
);

interface IInstallStatusMessage {
  installResult: ISoftwareScriptResult;
  isMyDevicePage: boolean;
  contactUrl?: string;
}

export const StatusMessage = ({
  installResult,
  isMyDevicePage,
  contactUrl,
}: IInstallStatusMessage) => {
  const {
    host_display_name,
    software_package,
    software_title,
    status,
    updated_at,
    created_at,
  } = installResult;

  const formattedHost = host_display_name ? (
    <b>{host_display_name}</b>
  ) : (
    "the host"
  );

  const displayTimeStamp = ["failed_install", "installed"].includes(
    status || ""
  )
    ? ` (${formatDistanceToNow(new Date(updated_at || created_at), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : "";

  const renderStatusCopy = () => {
    const prefix = (
      <>
        Fleet {getScriptDetailsStatusPredicate(status)} <b>{software_title}</b>
      </>
    );

    const middle = isMyDevicePage ? (
      <>
        {" "}
        {displayTimeStamp}
        {status === "failed_install" && (
          <>. You can rerun{renderContactOption(contactUrl)}</>
        )}
      </>
    ) : (
      <>
        {" "}
        ({software_package}) on {formattedHost}
        {status === "pending_install"
          ? " when it comes online"
          : displayTimeStamp}
      </>
    );
    return (
      <span>
        {prefix}
        {middle}
        {"."}
      </span>
    );
  };

  return (
    <div className={`${baseClass}__status-message`}>
      <Icon
        name={
          SCRIPT_DETAILS_STATUS_ICONS[status || "pending_install"] ??
          "pending-outline"
        }
      />
      {renderStatusCopy()}
    </div>
  );
};

interface IModalButtonsProps {
  deviceAuthToken?: string;
  installResultStatus?: string;
  hostSoftwareId?: number;
  onRerun?: (id: number) => void;
  onCancel: () => void;
}

export const ModalButtons = ({
  deviceAuthToken,
  installResultStatus,
  hostSoftwareId,
  onRerun,
  onCancel,
}: IModalButtonsProps) => {
  if (!!deviceAuthToken && installResultStatus === "failed_install") {
    const onClickRerun = () => {
      // on My Device Page, where this is relevant, both will be defined
      if (onRerun && hostSoftwareId) {
        onRerun(hostSoftwareId);
      }
      onCancel();
    };

    return (
      <ModalFooter
        primaryButtons={
          <>
            <Button variant="inverse" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" onClick={onClickRerun}>
              Rerun
            </Button>
          </>
        }
      />
    );
  }

  return (
    <ModalFooter primaryButtons={<Button onClick={onCancel}>Done</Button>} />
  );
};

interface ISoftwareInstallDetailsProps {
  /** note that details.install_uuid is present in hostSoftware, but since it is always needed for
  this modal while hostSoftware is not, as in the case of the activity feeds, it is specifically
  necessary in the details prop */
  details: IPackageInstallDetails;
  hostSoftware?: IHostSoftware; // for inventory versions, and software name when not Fleet installed (not present on activity feeds)
  deviceAuthToken?: string; // My Device Page only
  onCancel: () => void;
  onRerun?: (id: number) => void; // My Device Page only
  contactUrl?: string; // My Device Page only
}

export const SoftwareInstallDetailsModal = ({
  details: detailsFromProps,
  onCancel,
  hostSoftware,
  deviceAuthToken,
  onRerun,
  contactUrl,
}: ISoftwareInstallDetailsProps) => {
  // will always be present
  const installUUID = detailsFromProps.install_uuid ?? "";

  const [showInstallDetails, setShowInstallDetails] = useState(false);
  const toggleInstallDetails = () => {
    setShowInstallDetails((prev) => !prev);
  };

  const isInstalledByFleet = hostSoftware
    ? !!hostSoftware.software_package?.last_install
    : true; // if no hostSoftware passed in, can assume this is the activity feed, meaning this can only refer to a Fleet-handled install

  const { data: swInstallResult, isLoading, isError, error } = useQuery<
    ISoftwareInstallResults,
    AxiosError,
    ISoftwareScriptResult
  >(
    ["softwareInstallResults", installUUID],
    () => {
      return deviceAuthToken
        ? deviceUserAPI.getSoftwareInstallResult(deviceAuthToken, installUUID)
        : softwareAPI.getSoftwareInstallResult(installUUID);
    },
    {
      enabled: !!isInstalledByFleet,
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 3000,
      select: (data) => data.results as ISoftwareScriptResult,
    }
  );

  const renderScriptDetailsSection = () => (
    <>
      <RevealButton
        isShowing={showInstallDetails}
        showText="Details"
        hideText="Details"
        caretPosition="after"
        onClick={toggleInstallDetails}
      />
      {showInstallDetails && swInstallResult?.output && (
        <Textarea label="Install script output:" variant="code">
          {swInstallResult.output}
        </Textarea>
      )}
    </>
  );

  const hostDisplayname =
    swInstallResult?.host_display_name || detailsFromProps.host_display_name;

  const installResultWithHostDisplayName = swInstallResult
    ? {
        ...swInstallResult,
        host_display_name: hostDisplayname,
      }
    : undefined;

  const renderContent = () => {
    if (isInstalledByFleet) {
      if (isLoading) {
        return <Spinner />;
      }

      if (isError) {
        if (error?.status === 404) {
          return deviceAuthToken ? (
            <DeviceUserError />
          ) : (
            <DataError
              description="Couldn't get script details"
              excludeIssueLink
            />
          );
        }

        if (error?.status === 401) {
          return deviceAuthToken ? (
            <DeviceUserError />
          ) : (
            <DataError description="Close this modal and try again." />
          );
        }
      }

      if (!swInstallResult) {
        return deviceAuthToken ? (
          <DeviceUserError />
        ) : (
          <DataError description="No data returned." />
        );
      }

      if (
        !["installed", "pending_install", "failed_install"].includes(
          swInstallResult.status
        )
      ) {
        return (
          <DataError
            description={`Unexpected software install status ${swInstallResult.status}`}
          />
        );
      }
    }

    if (installResultWithHostDisplayName) {
      return (
        <div className={`${baseClass}__modal-content`}>
          <StatusMessage
            installResult={installResultWithHostDisplayName}
            isMyDevicePage={!!deviceAuthToken}
            contactUrl={contactUrl}
          />
          {renderScriptDetailsSection()}
        </div>
      );
    }
  };

  return (
    <Modal
      title="Script details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        {renderContent()}
        <ModalButtons
          deviceAuthToken={deviceAuthToken}
          installResultStatus={swInstallResult?.status}
          hostSoftwareId={hostSoftware?.id}
          onRerun={onRerun}
          onCancel={onCancel}
        />
      </>
    </Modal>
  );
};

export default SoftwareInstallDetailsModal;
