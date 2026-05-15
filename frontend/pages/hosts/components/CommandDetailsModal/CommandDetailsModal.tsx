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

export const GetIconName = (status: string): IconNames => {
  switch (status) {
    case "Error":
      return "error";
    case "CommandFormatError":
      return "error";
    case "Acknowledged":
      return "success";
    case "Pending":
      return "pending-outline";
    case "NotNow":
      return "pending-outline";

    default:
      // FIXME: update for other platforms and design appropriate default handling for unknown
      // statuses; for now, just return warning icon to indicate unknown state
      return "warning";
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

    default:
      // FIXME: update for other platforms and design appropriate default handling for unknown
      // statuses; for now, just fallback to status string
      return <span>{`Status: ${result.status}`}</span>;
  }
};

const defaultModalContentBody = (baseclass: string, result: ICommandResult) => (
  <IconStatusMessage
    className={`${baseclass}__status-message`}
    iconName={GetIconName(result.status)}
    message={getStatusMessage(result)}
  />
);

const ModalContent = ({
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
    // this should not happen, but just in case
    console.error("No results found in MDM command results data");
    return <DataError description="Close this modal and try again." />;
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
