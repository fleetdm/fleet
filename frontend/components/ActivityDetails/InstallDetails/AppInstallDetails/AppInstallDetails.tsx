// Used on: Dashboard > activity, Host details > past activity
// Also used on Self-service failed install details

import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { SoftwareInstallStatus } from "interfaces/software";
import mdmApi from "services/entities/mdm";
import deviceUserAPI from "services/entities/device_user";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import DeviceUserError from "components/DeviceUserError";
import Spinner from "components/Spinner/Spinner";
import { IMdmCommandResult } from "interfaces/mdm";
import { IActivityDetails } from "interfaces/activity";

import {
  getInstallDetailsStatusPredicate,
  INSTALL_DETAILS_STATUS_ICONS,
} from "../constants";

interface IGetStatusMessageProps {
  displayStatus: SoftwareInstallStatus | "pending";
  isStatusNotNow: boolean;
  isStatusAcknowledged: boolean;
  software_title: string;
  host_display_name: string;
}

export const getStatusMessage = ({
  displayStatus,
  isStatusNotNow,
  isStatusAcknowledged,
  software_title,
  host_display_name,
}: IGetStatusMessageProps) => {
  const formattedHost = host_display_name ? (
    <b>{host_display_name}</b>
  ) : (
    "the host"
  );

  // Handle NotNow case separately
  if (isStatusNotNow) {
    return (
      <>
        Fleet tried to install <b>{software_title}</b> on {formattedHost} but
        couldn&apos;t because the host was locked or was running on battery
        power while in Power Nap. Fleet will try again.
      </>
    );
  }

  // VPP Verify command pending state
  if (displayStatus === "pending_install" && isStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{software_title}</b> on{" "}
        {formattedHost} was acknowledged but the installation has not been
        verified. To re-check, select <b>Refetch</b> for this host.
      </>
    );
  }

  // Verification failed (timeout)
  if (displayStatus === "failed_install" && isStatusAcknowledged) {
    return (
      <>
        The MDM command (request) to install <b>{software_title}</b> on{" "}
        {formattedHost} was acknowledged but the installation has not been
        verified. Please re-attempt this installation.
      </>
    );
  }

  // Install command failed
  if (displayStatus === "failed_install") {
    return (
      <>
        The MDM command (request) to install <b>{software_title}</b> on{" "}
        {formattedHost} failed. Please re-attempt this installation.
      </>
    );
  }

  // Create predicate and subordinate for other statuses
  return (
    <>
      Fleet {getInstallDetailsStatusPredicate(displayStatus)}{" "}
      <b>{software_title}</b> on {formattedHost}
      {displayStatus === "pending" ? " when it comes online" : ""}.
    </>
  );
};

const baseClass = "app-install-details";

export type IAppInstallDetails = Pick<
  IActivityDetails,
  | "host_id"
  | "command_uuid"
  | "host_display_name"
  | "software_title"
  | "app_store_id"
  | "status"
> & {
  deviceAuthToken?: string;
};

export const AppInstallDetails = ({
  status,
  command_uuid = "",
  host_display_name = "",
  software_title = "",
  deviceAuthToken,
}: IAppInstallDetails) => {
  const { data: result, isLoading, isError, error } = useQuery<
    IMdmCommandResult,
    AxiosError
  >(
    ["mdm_command_results", command_uuid],
    async () => {
      return deviceAuthToken
        ? deviceUserAPI.getVppCommandResult(deviceAuthToken, command_uuid)
        : mdmApi.getCommandResults(command_uuid).then((response) => {
            const results = response.results?.[0];
            if (!results) {
              // FIXME: It's currently possible that the command results API response is empty for pending
              // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
              // display some minimal pending UI. This should be removed once the API response is fixed.
              return {} as IMdmCommandResult;
            }
            return {
              ...results,
              payload: atob(results.payload),
              result: atob(results.result),
            };
          });
    },
    {
      refetchOnWindowFocus: false,
      staleTime: 3000,
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
  } else if (!result) {
    // FIXME: It's currently possible that the command results API response is empty for pending
    // commands. As a temporary workaround to handle this case, we'll ignore the empty response and
    // display some minimal pending UI. This should be updated once the API response is fixed.
  }

  const displayStatus = (status as SoftwareInstallStatus) || "pending";
  const iconName = INSTALL_DETAILS_STATUS_ICONS[displayStatus];

  // Note: We need to reconcile status values from two different sources. From props, we
  // get the status from the activity item details (which can be "failed", "pending", or
  // "installed"). From the command results API response, we also receive the raw status
  // from the MDM protocol, e.g., "NotNow" or "Acknowledged". We need to display some special
  // messaging for the "NotNow" status, which otherwise would be treated as "pending".
  const isStatusNotNow = result?.status === "NotNow";
  const isStatusAcknowledged = result?.status === "Acknowledged";

  const statusMessage = getStatusMessage({
    displayStatus,
    isStatusNotNow,
    isStatusAcknowledged,
    software_title,
    host_display_name,
  });

  const showCommandPayload = !!result?.payload;
  const showCommandResponse =
    !!result?.result && (isStatusNotNow || status !== "pending");

  return (
    <>
      <div className={`${baseClass}__software-install-details`}>
        <div className={`${baseClass}__status-message`}>
          {!!iconName && <Icon name={iconName} />}
          <span>{statusMessage}</span>
        </div>
        {showCommandResponse && (
          <Textarea label="MDM command output:" variant="code">
            {result.result}
          </Textarea>
        )}
        {showCommandPayload && (
          <Textarea label="MDM command:" variant="code">
            {result.payload}
          </Textarea>
        )}
      </div>
    </>
  );
};

export const AppInstallDetailsModal = ({
  details,
  onCancel,
  deviceAuthToken,
}: {
  details: IAppInstallDetails;
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
          <AppInstallDetails deviceAuthToken={deviceAuthToken} {...details} />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};
