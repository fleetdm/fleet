import React from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { ICommand, ICommandResult } from "interfaces/command";

import commandApi, {
  IGetCommandResultsResponse,
  IGetHostCommandResultsQueryKey,
} from "services/entities/command";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import IconStatusMessage from "components/IconStatusMessage";
import { IconNames } from "components/icons";
import Textarea from "components/Textarea";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";

const baseClass = "command-details-modal";

const getIconName = (status: string): IconNames => {
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

  switch (result.status) {
    case "CommandFormatError":
    case "Error":
      return (
        <span>
          The <b>{result.request_type}</b> command failed on{" "}
          <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Acknowledged":
      return (
        <span>
          The <b>{result.request_type}</b> command ran on{" "}
          <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Pending":
      return (
        <span>
          The <b>{result.request_type}</b> command is running or will run on{" "}
          <b>{result.hostname}</b> when it comes online.
        </span>
      );

    case "NotNow":
      return (
        <span>
          The <b>{result.request_type}</b> command didn&apos;t run on{" "}
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

const ModalContent = ({
  data,
  isLoading,
  error,
}: {
  data: IGetCommandResultsResponse | undefined;
  isLoading: boolean;
  error: Error | null;
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
      <IconStatusMessage
        className={`${baseClass}__status-message`}
        iconName={getIconName(result.status)}
        message={getStatusMessage(result)}
      />
      {!!result.payload && (
        <Textarea label="Request payload:" variant="code">
          {result.payload}
        </Textarea>
      )}
      {!!result.result && (
        <Textarea
          label={
            <>
              Response from <b>{result.hostname}</b>:
            </>
          }
          variant="code"
        >
          {result.result}
        </Textarea>
      )}
    </div>
  );
};

interface ICommandResultsModalProps {
  command: ICommand;
  onDone: () => void;
}

const CommandResultsModal = ({
  command: { host_uuid: host_identifier, command_uuid },
  onDone,
}: ICommandResultsModalProps) => {
  const { data, isLoading, error } = useQuery<
    IGetCommandResultsResponse,
    Error,
    IGetCommandResultsResponse,
    IGetHostCommandResultsQueryKey[]
  >(
    [{ scope: "command_results", host_identifier, command_uuid }],
    ({ queryKey }) =>
      commandApi.getHostCommandResults(queryKey[0]).then((resp) => {
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
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      staleTime: 2000,
    }
  );

  return (
    <Modal
      className={baseClass}
      width="large"
      title="MDM command details"
      onExit={onDone}
    >
      <>
        <ModalContent data={data} isLoading={isLoading} error={error} />
        <ModalFooter primaryButtons={<Button onClick={onDone}>Done</Button>} />
      </>
    </Modal>
  );
};

export default CommandResultsModal;
