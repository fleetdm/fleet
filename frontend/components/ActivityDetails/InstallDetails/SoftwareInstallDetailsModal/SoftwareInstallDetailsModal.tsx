/** For payload-free packages (e.g. software source is sh_packages or ps1_packages)
 * we use SoftwareScriptDetailsModal
 * For iOS/iPadOS packages (e.g. .ipa packages software source is ios_apps or ipados_apps)
 * we use SoftwareIpaInstallDetailsModal with the command_uuid */

import React, { useState } from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";
import { AxiosError } from "axios";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import {
  IHostSoftware,
  ISoftwareInstallResult,
  ISoftwareInstallResults,
} from "interfaces/software";
import softwareAPI from "services/entities/software";
import deviceUserAPI from "services/entities/device_user";

import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import IconStatusMessage from "components/IconStatusMessage";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner/Spinner";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";

import {
  INSTALL_DETAILS_STATUS_ICONS,
  getInstallDetailsStatusPredicate,
} from "../constants";

const baseClass = "software-install-details-modal";

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
  softwareName: string;
  installResult?: ISoftwareInstallResult;
  isMyDevicePage: boolean;
  contactUrl?: string;
}

// TODO - match VppInstallDetailsModal status to this, still accounting for MDM-specific cases
// present there
export const StatusMessage = ({
  softwareName,
  installResult,
  isMyDevicePage,
  contactUrl,
}: IInstallStatusMessage) => {
  // the case when software is installed by the user and not by Fleet
  if (!installResult) {
    return (
      <IconStatusMessage
        className={`${baseClass}__status-message`}
        iconName="success"
        message={
          <span>
            <b>{softwareName}</b> is installed.
          </span>
        }
      />
    );
  }

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
        Fleet {getInstallDetailsStatusPredicate(status)} <b>{software_title}</b>
      </>
    );

    const middle = isMyDevicePage ? (
      <>
        {" "}
        {displayTimeStamp}
        {status === "failed_install" && (
          <>. You can retry{renderContactOption(contactUrl)}</>
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
    <IconStatusMessage
      className={`${baseClass}__status-message`}
      iconName={
        INSTALL_DETAILS_STATUS_ICONS[status || "pending_install"] ??
        "pending-outline"
      }
      message={renderStatusCopy()}
    />
  );
};

interface IModalButtonsProps {
  deviceAuthToken?: string;
  status?: string;
  hostSoftwareId?: number;
  onRetry?: (id: number) => void;
  onCancel: () => void;
}

export const ModalButtons = ({
  deviceAuthToken,
  status,
  hostSoftwareId,
  onRetry,
  onCancel,
}: IModalButtonsProps) => {
  if (deviceAuthToken && status === "failed_install") {
    const onClickRetry = () => {
      // on My Device Page, where this is relevant, both will be defined
      if (onRetry && hostSoftwareId) {
        onRetry(hostSoftwareId);
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
            <Button type="submit" onClick={onClickRetry}>
              Retry
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
  onRetry?: (id: number) => void; // My Device Page only
  contactUrl?: string; // My Device Page only
}

export const SoftwareInstallDetailsModal = ({
  details: detailsFromProps,
  onCancel,
  hostSoftware,
  deviceAuthToken,
  onRetry,
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
    ISoftwareInstallResult
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
      select: (data) => data.results,
    }
  );

  const renderInventoryVersionsSection = () => {
    if (hostSoftware?.installed_versions?.length) {
      return <InventoryVersions hostSoftware={hostSoftware} />;
    }
    return "If you uninstalled it outside of Fleet it will still show as installed.";
  };

  const renderInstallDetailsSection = () => {
    const outputs = [
      {
        label: "Install script output:",
        value: swInstallResult?.output,
      },
      {
        label: "Post-install script output:",
        value: swInstallResult?.post_install_script_output,
      },
    ];

    return (
      <>
        <RevealButton
          isShowing={showInstallDetails}
          showText="Details"
          hideText="Details"
          caretPosition="after"
          onClick={toggleInstallDetails}
        />
        {showInstallDetails &&
          outputs.map(
            ({ label, value }) =>
              value && (
                <Textarea key={label} label={label} variant="code">
                  {value}
                </Textarea>
              )
          )}
      </>
    );
  };

  const excludeVersions = ["pending_install", "failed_install"].includes(
    swInstallResult?.status || ""
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
              description="Couldn't get install details"
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

    return (
      <div className={`${baseClass}__modal-content`}>
        <StatusMessage
          installResult={installResultWithHostDisplayName}
          softwareName={hostSoftware?.name || "Software"} // will always be defined at this point
          isMyDevicePage={!!deviceAuthToken}
          contactUrl={contactUrl}
        />

        {hostSoftware && !excludeVersions && renderInventoryVersionsSection()}

        {swInstallResult?.status !== "pending_install" &&
          isInstalledByFleet &&
          renderInstallDetailsSection()}
      </div>
    );
  };

  return (
    <Modal
      title="Install details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        {renderContent()}
        <ModalButtons
          deviceAuthToken={deviceAuthToken}
          status={swInstallResult?.status}
          hostSoftwareId={hostSoftware?.id}
          onRetry={onRetry}
          onCancel={onCancel}
        />
      </>
    </Modal>
  );
};

export default SoftwareInstallDetailsModal;
