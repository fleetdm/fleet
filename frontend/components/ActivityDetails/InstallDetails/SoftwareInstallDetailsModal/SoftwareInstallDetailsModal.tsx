import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";
import { AxiosError } from "axios";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { AppContext } from "context/app";

import { IActivityDetails } from "interfaces/activity";
import {
  IHostSoftware,
  ISoftwareInstallResult,
  ISoftwareInstallResults,
  ISoftwareInstallVersion,
  ISoftwareTitleDetails,
  SoftwareSource,
} from "interfaces/software";
import softwareAPI, {
  IGetSoftwareTitleQueryKey,
  ISoftwareTitleResponse,
} from "services/entities/software";
import deviceUserAPI from "services/entities/device_user";

import InventoryVersions from "pages/hosts/details/components/InventoryVersions";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
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
  deviceAuthToken?: string;
};

const StatusMessage = ({
  result: {
    host_display_name,
    software_package,
    software_title,
    status,
    updated_at,
    created_at,
  },
  isDUP,
  isInstalledByFleet,
}: {
  result: ISoftwareInstallResult;
  isDUP: boolean;
  isInstalledByFleet: boolean;
}) => {
  const { config } = useContext(AppContext);

  const formattedHost = host_display_name ? (
    <b>{host_display_name}</b>
  ) : (
    "the host"
  );

  // TODO: Potential implementation HumanTimeDiffWithDateTip for consistency
  // Design currently looks weird since displayTimeStamp might split to multiple lines
  const timeStamp = updated_at || created_at;
  const displayTimeStamp = ["failed_install", "installed"].includes(
    status || ""
  )
    ? ` (${formatDistanceToNow(new Date(timeStamp), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : "";

  const renderContactOption = () => (
    // TODO - config is undefined in the DUP context, will need to get this contact_url from
    // somewhere else or omit the link
    <>
      {" "}
      or{" "}
      {config?.org_info.contact_url ? (
        <CustomLink
          url={config.org_info.contact_url}
          text="contact your IT admin"
          newTab
        />
      ) : (
        "contact your IT admin"
      )}
    </>
  );

  const renderStatusCopy = () => {
    if (isInstalledByFleet) {
      const prefix = (
        <>
          Fleet {getInstallDetailsStatusPredicate(status)}{" "}
          <b>{software_title}</b>
        </>
      );
      let middle = null;
      if (isDUP) {
        if (status === "failed_install") {
          middle = <>. You can retry{renderContactOption()}</>;
        }
      } else {
        // host details page
        middle = (
          <>
            {" "}
            ({software_package}) on {formattedHost}
            {status === "pending_install" ? " when it comes online" : ""}
            {displayTimeStamp}
          </>
        );
      }
      return (
        <span>
          {prefix}
          {middle}
          {"."}
        </span>
      );
    }
    if (status === "installed") {
      return (
        <span>
          <b>{software_title}</b> is installed.
        </span>
      );
    }
    return (
      <DataError description="Bad data from server - software cannot be both not installed (failed/pending) and not installed by Fleet" />
    );
  };

  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={INSTALL_DETAILS_STATUS_ICONS[status] ?? "pending-outline"} />
      {renderStatusCopy()}
    </div>
  );
};

interface ISoftwareInstallDetailsProps {
  // note that details.install_uuid is present in hostSoftware, but since it is always needed for
  // this modal while hostSoftware is not, as in the case of the activity feeds, it is specifically
  // necessary in the details prop
  details: IPackageInstallDetails;
  onCancel: () => void;
  hostSoftware?: IHostSoftware; // for inventory versions, not present on activity feeds
  deviceAuthToken?: string;
  onClickRetry?: () => void; // DUP only
}

export const SoftwareInstallDetailsModal = ({
  details: detailsFromProps,
  onCancel,
  hostSoftware,
  deviceAuthToken,
  onClickRetry,
}: ISoftwareInstallDetailsProps) => {
  // will always be present
  const installUUID = detailsFromProps.install_uuid ?? "";

  const [showInstallDetails, setShowInstallDetails] = useState(false);
  const toggleInstallDetails = () => {
    setShowInstallDetails((prev) => !prev);
  };

  // TODO - VPP case (see AppInstallDetails)

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
      refetchOnWindowFocus: false,
      staleTime: 3000,
      select: (data) => data.results,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
    }
  );

  // need to fetch the install script
  // TODO - include in above response if possible to avoid this secondary API call
  const {
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
  } = useQuery<
    ISoftwareTitleResponse,
    AxiosError,
    ISoftwareTitleDetails,
    IGetSoftwareTitleQueryKey[]
  >(
    [
      {
        scope: "softwareById",
        softwareId: swInstallResult?.software_title_id || 0,
      },
    ],
    ({ queryKey }) => softwareAPI.getSoftwareTitle(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.software_title,
    }
  );

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    if (error?.status === 404) {
      return deviceAuthToken ? (
        <DeviceUserError />
      ) : (
        <DataError
          description="Install details are no longer available for this activity."
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
    // FIXME: Find a better solution for this.
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

  const renderInstallDetails = () => {
    const renderScript = () => {
      if (isSoftwareTitleLoading) {
        return <Spinner />;
      }
      if (isSoftwareTitleError) {
        return (
          <DataError
            description="Unable to fetch install script for this software."
            excludeIssueLink
          />
        );
      }
      return (
        <Textarea label="Install script:" variant="code">
          {softwareTitle?.software_package?.install_script}
        </Textarea>
      );
    };
    return (
      <>
        {swInstallResult.output && (
          <Textarea label="Install script output:" variant="code">
            {swInstallResult.output}
          </Textarea>
        )}
        {renderScript()}
      </>
    );
  };

  const renderInstallDetailsSection = () => {
    return (
      <>
        <RevealButton
          isShowing={showInstallDetails}
          showText="Details"
          hideText="Details"
          caretPosition="after"
          onClick={toggleInstallDetails}
        />
        {showInstallDetails && renderInstallDetails()}
      </>
    );
  };

  const isInstalledByFleet = hostSoftware
    ? !!hostSoftware.software_package?.last_install
    : true; // if no hostSoftware passed in, can assume this is the activity feed, meaning this can only refer to a Fleet-handled install

  const excludeVersions =
    !deviceAuthToken &&
    ["pending_install", "failed_install"].includes(swInstallResult.status);

  const host_display_name =
    swInstallResult.host_display_name || detailsFromProps.host_display_name;

  const renderCta = () => {
    if (deviceAuthToken && swInstallResult.status === "failed_install") {
      return (
        <div className="modal-cta-wrap">
          <Button type="submit" onClick={onClickRetry}>
            Retry
          </Button>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      );
    }
    return (
      <div className="modal-cta-wrap">
        <Button onClick={onCancel}>Done</Button>
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
        <div className={`${baseClass}__modal-content`}>
          <StatusMessage
            result={{ ...swInstallResult, host_display_name }}
            isDUP={!!deviceAuthToken}
            isInstalledByFleet={isInstalledByFleet}
          />

          {hostSoftware && !excludeVersions && (
            <InventoryVersions hostSoftware={hostSoftware} />
          )}

          {swInstallResult.status !== "pending_install" &&
            isInstalledByFleet &&
            renderInstallDetailsSection()}
        </div>
        {renderCta()}
      </>
    </Modal>
  );
};

export default SoftwareInstallDetailsModal;
