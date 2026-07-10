import React from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { ICommandResult } from "interfaces/command";

import commandApi, {
  IGetCommandResultsResponse,
  IGetHostCommandResultsQueryKey,
} from "services/entities/command";

import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import IconStatusMessage from "components/IconStatusMessage";
import { IconNames } from "components/icons";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";

const baseClass = "command-details-modal";

export const getIconName = (status: string): IconNames => {
  // Apple MDM status strings
  switch (status) {
    case "Error":
    case "CommandFormatError":
      return "error";
    case "Acknowledged":
      return "success";
    case "Pending":
    case "NotNow":
      return "pending-outline";
    // sentinel used when the command results API returns a 200 with no
    // results (e.g. the host it was sent to was wiped and re-enrolled since)
    case "Deleted":
      return "info-outline";
    default:
      break;
  }
  // Windows OMA-DM status codes (numeric strings): 101 = pending, 200-399 = ran, 400+ = failed
  const code = parseInt(status, 10);
  if (!Number.isNaN(code)) {
    if (code >= 400) return "error";
    if (code >= 200) return "success";
    return "pending-outline";
  }
  return "warning";
};

export const getVerbForCommandStatus = (status: string): string => {
  const icon = getIconName(status);
  switch (icon) {
    case "error":
      return "failed to run";
    case "success":
      return "ran";
    case "pending-outline":
      return "sent";
    default:
      // unknown status
      return "sent";
  }
};

const getStatusMessage = (result: ICommandResult): React.ReactNode => {
  const displayTime = result.updated_at
    ? ` (${formatDistanceToNow(new Date(result.updated_at), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : null;

  const namePart = result.name ? (
    <>
      {" "}
      for <b>{result.name}</b>
    </>
  ) : null;

  switch (result.status) {
    case "CommandFormatError":
    case "Error":
      return (
        <span>
          The <b>{result.request_type}</b> command{namePart} failed on{" "}
          <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Acknowledged":
      return (
        <span>
          The <b>{result.request_type}</b> command{namePart} was acknowledged by{" "}
          <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Pending":
      return (
        <span>
          The <b>{result.request_type}</b> command{namePart} is pending on{" "}
          <b>{result.hostname}</b>.
        </span>
      );

    case "NotNow":
      return (
        <span>
          The <b>{result.request_type}</b> command{namePart} is deferred on{" "}
          <b>{result.hostname}</b> because the host was locked or was running on
          battery power while in Power Nap. Fleet will try again.
        </span>
      );

    case "Deleted":
      return <span>This command has been deleted.</span>;

    default:
      // FIXME: update for other platforms and design appropriate default handling for unknown
      // statuses; for now, just fallback to status string
      return <span>{`Status: ${result.status}`}</span>;
  }
};

const defaultModalContentBody = (baseclass: string, result: ICommandResult) => (
  <IconStatusMessage
    className={`${baseclass}__status-message`}
    iconName={getIconName(result.status)}
    message={getStatusMessage(result)}
  />
);

export const ModalContent = ({
  data,
  isLoading,
  error,
  contentBody = defaultModalContentBody,
}: {
  data: IGetCommandResultsResponse | undefined;
  isLoading: boolean;
  error: Error | null;
  contentBody?: (baseClass: string, result: ICommandResult) => React.ReactNode;
}) => {
  if (isLoading) {
    return <Spinner />;
  }

  if (error) {
    return <DataError description="Close this modal and try again." />;
  }

  if (!data?.results?.[0]) {
    // a 200 with no results means the command no longer has anything to show --
    // most commonly because the host it was sent to was wiped and re-enrolled
    // since. Render the modal normally (via the caller's contentBody, same as a
    // real result) rather than as an error, since nothing actually went wrong.
    // The "Deleted" sentinel status lets the caller render its own copy for
    // this case using the activity's own details, since there's no real
    // result to pull hostname/request_type from.
    const deletedCommandResult: ICommandResult = {
      host_uuid: "",
      command_uuid: "",
      status: "Deleted",
      updated_at: "",
      request_type: "",
      hostname: "",
      payload: "",
      result: "",
      name: null,
    };
    return (
      <div className={`${baseClass}__modal-content`}>
        {contentBody(baseClass, deletedCommandResult)}
      </div>
    );
  }

  if (data.results.length > 1) {
    // this should not happen, but just in case
    console.error(
      `Expected one result, but found ${data.results.length} results.`
    );
    return <DataError description="Close this modal and try again." />;
  }

  const result = data.results[0];

  return (
    <div className={`${baseClass}__modal-content`}>
      {contentBody(baseClass, result)}
      {!!result.payload && (
        <InputField
          type="textarea"
          label="Request payload:"
          value={result.payload}
          readOnly
          enableCopy
        />
      )}
      {!!result.result && (
        <InputField
          type="textarea"
          label={
            <>
              Response from <b>{result.hostname}</b>:
            </>
          }
          value={result.result}
          readOnly
          enableCopy
        />
      )}
    </div>
  );
};

type ICommandResultsModalCommand = {
  host_uuid?: string;
  command_uuid: string;
};

interface ICommandResultsModalProps {
  command: ICommandResultsModalCommand;
  // contentBody if provided will be used to render content above the request and response payloads.
  // if not defined, a default contentBody will be used to display a status message and icon based on profile status
  contentBody?: (baseClass: string, result: ICommandResult) => React.ReactNode;
  title?: string;
  onDone: () => void;
}

const CommandResultsModal = ({
  command: { host_uuid: host_identifier, command_uuid },
  contentBody,
  title = "MDM command details",
  onDone,
}: ICommandResultsModalProps) => {
  const { data, isLoading, error } = useQuery<
    IGetCommandResultsResponse,
    Error,
    IGetCommandResultsResponse,
    IGetHostCommandResultsQueryKey[]
  >(
    [
      {
        scope: "command_results",
        host_identifier: host_identifier ?? "",
        command_uuid,
      },
    ],
    async ({ queryKey }) => {
      const resp =
        queryKey[0].host_identifier === ""
          ? // if host_identifier is not provided, use the getCommandResults endpoint which does not require host_identifier
            await commandApi.getCommandResults(queryKey[0].command_uuid)
          : await commandApi.getHostCommandResults(queryKey[0]);

      if (!resp?.results) {
        // this should not happen, but just in case return the response as is
        return resp;
      }
      return {
        results: resp.results.map?.((r) => ({
          ...r,
          payload: atob(r.payload),
          result: atob(r.result),
        })),
      };
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  return (
    <Modal className={baseClass} width="large" title={title} onExit={onDone}>
      <ModalContent
        data={data}
        isLoading={isLoading}
        error={error}
        contentBody={contentBody}
      />
      <ModalFooter primaryButtons={<Button onClick={onDone}>Close</Button>} />
    </Modal>
  );
};

export default CommandResultsModal;
