// Used on: Dashboard > activity, Host details > past activity
// Also used on Self-service failed install details

import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";
import { AxiosError } from "axios";

import { AppContext } from "context/app";

import { IActivityDetails } from "interfaces/activity";
import {
  ISoftwareInstallResult,
  ISoftwareInstallResults,
  ISoftwareInstallVersion,
  SoftwareSource,
} from "interfaces/software";
import softwareAPI from "services/entities/software";
import deviceUserAPI from "services/entities/device_user";

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

// TODO: Expand to include more details as needed
export type IPackageInstallDetails = Pick<
  IActivityDetails,
  "install_uuid" | "host_display_name"
> & {
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

  const renderContactOption = () =>
    config?.org_info.contact_url ? (
      <>
        {" "}
        or{" "}
        <CustomLink
          url={config.org_info.contact_url}
          text="contact your IT admin"
          newTab
        />
      </>
    ) : null;

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
          middle = <>You can retry{renderContactOption()}</>;
        }
      } else {
        // host details page
        middle = (
          <>
            {" "}
            ({software_package}) on {formattedHost}
            {status === "pending_install" ? " when it comes online" : ""}
            {/* TODO - need to add "about" before timestamp? */}
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
      // TODO - is this check necessary, i.e. is it possible to have pending/failed software that
      // was not installed by Fleet?
      return (
        <span>
          <b>{software_title}</b> is installed.
        </span>
      );
    }
    return (
      <DataError description="Software cannot be both not installed (failed/pending) and not installed by Fleet" />
    );
  };

  return (
    <div className={`${baseClass}__status-message`}>
      <Icon name={INSTALL_DETAILS_STATUS_ICONS[status] ?? "pending-outline"} />
      {renderStatusCopy()}
    </div>
  );
};

const SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS = {
  pre_install_query_output: "Pre-install condition",
  output: "Software install output",
  post_install_script_output: "Post-install script output",
} as const;

const Output = ({
  displayKey,
  result,
}: {
  displayKey: keyof typeof SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS;
  result: ISoftwareInstallResult;
}) => {
  return (
    <Textarea
      label={`${SOFTWARE_INSTALL_OUTPUT_DISPLAY_LABELS[displayKey]}:`}
      variant="code"
    >
      {result[displayKey]}
    </Textarea>
  );
};

// TODO - TURN THIS INTO INVENTORY VERSIONS
interface IInventoryVersionsProps {
  installedVersion: ISoftwareInstallVersion;
  source: SoftwareSource;
  bundleIdentifier?: string;
}
const InventoryVersions = ({
  installedVersion,
  source,
  bundleIdentifier,
}: IInventoryVersionsProps) => {
  const {
    vulnerabilities,
    installed_paths: installedPaths,
    signature_information: signatureInformation,
  } = installedVersion;

  // return (
  //   <Card
  //     className={`${baseClass}__version-details`}
  //     color="grey"
  //     borderRadiusSize="medium"
  //   >
  //     <div className={`${baseClass}__row`}>
  //       <DataSet title="Version" value={installedVersion.version} />
  //       <DataSet title="Type" value={formatSoftwareType({ source })} />
  //       {bundleIdentifier && (
  //         <DataSet title="Bundle identifier" value={bundleIdentifier} />
  //       )}
  //       {installedVersion.last_opened_at && (
  //         <DataSet
  //           title="Last used"
  //           value={dateAgo(installedVersion.last_opened_at)}
  //         />
  //       )}
  //     </div>
  //     {vulnerabilities && vulnerabilities.length !== 0 && (
  //       <div className={`${baseClass}__row`}>
  //         <DataSet
  //           title="Vulnerabilities"
  //           value={generateVulnerabilitiesValue(vulnerabilities)}
  //         />
  //       </div>
  //     )}
  //     {!!installedPaths?.length &&
  //       installedPaths.map((path) => {
  //         // Find the signature info for this path
  //         const sigInfo = signatureInformation?.find(
  //           (info) => info.installed_path === path
  //         );

  //         return (
  //           <div className={`${baseClass}__sig-info`}>
  //             <DataSet orientation="horizontal" title="Path" value={path} />
  //             {sigInfo?.hash_sha256 && (
  //               <DataSet
  //                 orientation="horizontal"
  //                 title="Hash"
  //                 value={sigInfo.hash_sha256}
  //               />
  //             )}
  //           </div>
  //         );
  //       })}
  //   </Card>
  // );
};

// TODO - remove this layer of abstraction, no longer necessary
export const SoftwareInstallDetails = ({
  host_display_name = "",
  install_uuid = "",
  deviceAuthToken,
}: IPackageInstallDetails) => {
  const [showInstallDetails, setShowInstallDetails] = useState(false);
  const toggleInstallDetails = () => {
    setShowInstallDetails((prev) => !prev);
  };
  const { data: result, isLoading, isError, error } = useQuery<
    ISoftwareInstallResults,
    AxiosError,
    ISoftwareInstallResult
  >(
    ["softwareInstallResults", install_uuid],
    () => {
      return deviceAuthToken
        ? deviceUserAPI.getSoftwareInstallResult(deviceAuthToken, install_uuid)
        : softwareAPI.getSoftwareInstallResult(install_uuid);
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
      select: (data) => data.results,
      retry: (failureCount, err) => err?.status !== 404 && failureCount < 3,
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

  if (!result) {
    // FIXME: Find a better solution for this.
    return deviceAuthToken ? (
      <DeviceUserError />
    ) : (
      <DataError description="No data returned." />
    );
  }

  if (
    !["installed", "pending_install", "failed_install"].includes(result.status)
  ) {
    return (
      <DataError
        description={`Unexpected software install status ${result.status}`}
      />
    );
  }

  const renderInstallDetails = () => (
    <>
      {/* {result.pre_install_query_output && (
        <Output displayKey="pre_install_query_output" result={result} />
      )}
      {result.output && <Output displayKey="output" result={result} />}
      {result.post_install_script_output && (
        <Output displayKey="post_install_script_output" result={result} />
      )} */}
      {result.output && (
        <Textarea label="Install script output:" variant="code">
          {result.output}
        </Textarea>
      )}
      <Textarea label="Install script:" variant="code">
        {/* TODO get the install script! */}
        {"TODO - get the install script!"}
      </Textarea>
    </>
  );

  const renderInstallDetailsSection = () => {
    if (!deviceAuthToken && result.status === "failed_install") {
      return renderInstallDetails();
    }
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

  // QUESTION - how to determine if SW is installed by Fleet?
  // const isInstalledByFleet = false;
  const isInstalledByFleet = true;

  const excludeVersions =
    !deviceAuthToken &&
    ["pending_install", "failed_install"].includes(result.status);

  return (
    <>
      <StatusMessage
        result={
          result.host_display_name ? result : { ...result, host_display_name } // prefer result.host_display_name (it may be empty if the host was deleted) otherwise default to whatever we received via props
        }
        isDUP={!!deviceAuthToken}
        isInstalledByFleet={isInstalledByFleet}
      />

      {/* TODO - flesh out */}
      {/* {!excludeVersions && <InventoryVersions />} */}

      {result.status !== "pending_install" &&
        isInstalledByFleet &&
        renderInstallDetailsSection()}
    </>
  );
};

export const SoftwareInstallDetailsModal = ({
  details,
  onCancel,
  deviceAuthToken,
}: {
  details: IPackageInstallDetails;
  onCancel: () => void;
  deviceAuthToken?: string;
}) => {
  return (
    <Modal
      title="Install details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <SoftwareInstallDetails
            {...details}
            deviceAuthToken={deviceAuthToken}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};
