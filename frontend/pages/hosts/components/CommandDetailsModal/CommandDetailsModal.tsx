import React, { useCallback, useState } from "react";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { stringToClipboard } from "utilities/copy_text";

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
import Icon from "components/Icon";
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

const isProfileCommand = (requestType: string): boolean =>
  requestType === "InstallProfile" || requestType === "RemoveProfile";

const getStatusMessage = (result: ICommandResult): React.ReactNode => {
  const displayTime = result.updated_at
    ? ` (${formatDistanceToNow(new Date(result.updated_at), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : null;

  const profileNamePart =
    isProfileCommand(result.request_type) && result.name ? (
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
          The <b>{result.request_type}</b> command{profileNamePart} failed on{" "}
          <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Acknowledged":
      return (
        <span>
          The <b>{result.request_type}</b> command{profileNamePart} was
          acknowledged by <b>{result.hostname}</b>
          {displayTime}.
        </span>
      );

    case "Pending":
      return (
        <span>
          The <b>{result.request_type}</b> command{profileNamePart} is pending
          on <b>{result.hostname}</b>.
        </span>
      );

    case "NotNow":
      return (
        <span>
          The <b>{result.request_type}</b> command{profileNamePart} is deferred
          on <b>{result.hostname}</b> because the host was locked or was running
          on battery power while in Power Nap. Fleet will try again.
        </span>
      );

    default:
      // FIXME: update for other platforms and design appropriate default handling for unknown
      // statuses; for now, just fallback to status string
      return <span>{`Status: ${result.status}`}</span>;
  }
};

/** Formats a command result into a text representation suitable for
 * clipboard copy (equivalent to `fleetctl get mdm-command-results --id ...`). */
const formatCommandDetailsForCopy = (result: ICommandResult): string => {
  const lines: string[] = [];
  lines.push(`Host UUID: ${result.host_uuid}`);
  lines.push(`Command UUID: ${result.command_uuid}`);
  lines.push(`Status: ${result.status}`);
  lines.push(`Request type: ${result.request_type}`);
  if (result.name) {
    lines.push(`Name: ${result.name}`);
  }
  lines.push(`Updated: ${result.updated_at}`);
  lines.push(`Hostname: ${result.hostname}`);
  if (result.payload) {
    lines.push("");
    lines.push("--- Request payload ---");
    lines.push(result.payload);
  }
  if (result.result) {
    lines.push("");
    lines.push(`--- Response from ${result.hostname} ---`);
    lines.push(result.result);
  }
  return lines.join("\n");
};

type CopyTarget = "payload" | "response" | "details";

const CopyButton = ({
  text,
  target,
  copyState,
  onCopy,
}: {
  text: string;
  target: CopyTarget;
  copyState: CopyTarget | null;
  onCopy: (value: string, target: CopyTarget) => void;
}) => {
  const isCopied = copyState === target;

  return (
    <div className={`${baseClass}__copy-wrapper`}>
      {isCopied && (
        <span className={`${baseClass}__copied-confirmation`}>Copied!</span>
      )}
      <Button
        variant="icon"
        onClick={(e: React.MouseEvent) => {
          e.preventDefault();
          onCopy(text, target);
        }}
        iconStroke
      >
        <Icon name="copy" />
      </Button>
    </div>
  );
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
  const [copyState, setCopyState] = useState<CopyTarget | null>(null);

  const onCopy = useCallback((value: string, target: CopyTarget) => {
    stringToClipboard(value).then(() => {
      setCopyState(target);
      setTimeout(() => {
        setCopyState(null);
      }, 2000);
    });
  }, []);

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
        <div className={`${baseClass}__section`}>
          <div className={`${baseClass}__section-header`}>
            <div className="textarea__label">Request payload:</div>
            <CopyButton
              text={result.payload}
              target="payload"
              copyState={copyState}
              onCopy={onCopy}
            />
          </div>
          <Textarea variant="code">{result.payload}</Textarea>
        </div>
      )}
      {!!result.result && (
        <div className={`${baseClass}__section`}>
          <div className={`${baseClass}__section-header`}>
            <div className="textarea__label">
              Response from <b>{result.hostname}</b>:
            </div>
            <CopyButton
              text={result.result}
              target="response"
              copyState={copyState}
              onCopy={onCopy}
            />
          </div>
          <Textarea variant="code">{result.result}</Textarea>
        </div>
      )}
      <div className={`${baseClass}__copy-all`}>
        <Button
          variant="text-link"
          onClick={(e: React.MouseEvent) => {
            e.preventDefault();
            onCopy(formatCommandDetailsForCopy(result), "details");
          }}
        >
          <Icon name="copy" />
          {copyState === "details" ? "Copied!" : "Copy command details"}
        </Button>
      </div>
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
      <ModalContent data={data} isLoading={isLoading} error={error} />
      <ModalFooter primaryButtons={<Button onClick={onDone}>Done</Button>} />
    </Modal>
  );
};

export default CommandResultsModal;
