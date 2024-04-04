import React from "react";
import { useQuery } from "react-query";

import scriptsAPI, { IScriptResultResponse } from "services/entities/scripts";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Textarea from "components/Textarea";
import DataError from "components/DataError/DataError";
import Spinner from "components/Spinner/Spinner";

const baseClass = "script-details-modal";

interface IScriptContentProps {
  content: string;
}

const ScriptContent = ({ content }: IScriptContentProps) => {
  return (
    <div className={`${baseClass}__script-content`}>
      <span>Script content:</span>
      <Textarea className={`${baseClass}__script-content-textarea`}>
        {content}
      </Textarea>
    </div>
  );
};

const StatusMessageRunning = () => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="pending-outline" />
      Script is running or will run when the host comes online.
    </p>
  </div>
);

const StatusMessageSuccess = () => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="success-outline" />
      Exit code: 0 (Script ran successfully.)
    </p>
  </div>
);

const StatusMessageFailed = ({ exitCode }: { exitCode: number }) => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="error-outline" />
      Exit code: {exitCode} (Script failed.)
    </p>{" "}
  </div>
);

const StatusMessageError = ({ message }: { message: string }) => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="error-outline" />
      Error: {message}
    </p>
  </div>
);

interface IStatusMessageProps {
  hostTimeout: boolean;
  exitCode: number | null;
  message: string;
}

const StatusMessage = ({
  hostTimeout,
  exitCode,
  message,
}: IStatusMessageProps) => {
  switch (exitCode) {
    case null:
      return !hostTimeout ? (
        // Expected API message: "A script is already running on this host. Please wait about 1 minute to let it finish."
        <StatusMessageRunning />
      ) : (
        // Expected API message: "Fleet hasn’t heard from the host in over 1 minute. Fleet doesn’t know if the script ran because the host went offline."
        <StatusMessageError message={message} />
      );
    case -2:
      // Expected API message: "Scripts are disabled for this host. To run scripts, deploy the fleetd agent with scripts enabled."
      return <StatusMessageError message={message} />;
    case -1:
      // Expected API message: "Timeout. Fleet stopped the script after 5 minutes to protect host performance."
      return <StatusMessageError message={message} />;
    case 0:
      // Expected API message: ""
      return <StatusMessageSuccess />;
    default:
      // Expected API message: ""
      return <StatusMessageFailed exitCode={exitCode} />;
  }
};

interface IScriptOutputProps {
  output: string;
  hostname: string;
}

const ScriptOutput = ({ output, hostname }: IScriptOutputProps) => {
  return (
    <div className={`${baseClass}__script-output`}>
      <p>
        The{" "}
        <TooltipWrapper
          tipContent="Fleet records the last 10,000 characters to prevent downtime."
          tooltipClass={`${baseClass}__output-tooltip`}
          isDelayed
        >
          output recorded
        </TooltipWrapper>{" "}
        when <b>{hostname}</b> ran the script above:
      </p>
      <Textarea className={`${baseClass}__output-textarea`}>{output}</Textarea>
    </div>
  );
};

interface IScriptResultProps {
  hostname: string;
  hostTimeout: boolean;
  exitCode: number | null;
  message: string;
  output: string;
}

const ScriptResult = ({
  hostname,
  hostTimeout,
  exitCode,
  message,
  output,
}: IScriptResultProps) => {
  const hostTimedOut = exitCode === null && hostTimeout === true;
  const scriptsDisabledForHost = exitCode === -2;
  const scriptStillRunning = exitCode === null && hostTimeout === false;
  const showOutputText =
    !hostTimedOut && !scriptsDisabledForHost && !scriptStillRunning;

  return (
    <div className={`${baseClass}__script-result`}>
      <StatusMessage
        hostTimeout={hostTimeout}
        exitCode={exitCode}
        message={message}
      />
      {showOutputText && <ScriptOutput output={output} hostname={hostname} />}
    </div>
  );
};

interface IScriptDetailsModalProps {
  scriptExecutionId: string;
  onCancel: () => void;
}

const ScriptDetailsModal = ({
  scriptExecutionId,
  onCancel,
}: IScriptDetailsModalProps) => {
  const { data, isLoading, isError } = useQuery<IScriptResultResponse>(
    ["scriptDetailsModal", scriptExecutionId],
    () => {
      return scriptsAPI.getScriptResult(scriptExecutionId);
    },
    { refetchOnWindowFocus: false }
  );

  const renderContent = () => {
    let content = <></>;

    if (isLoading) {
      content = <Spinner />;
    } else if (isError) {
      content = <DataError description="Close this modal and try again." />;
    } else if (data) {
      content = (
        <>
          <ScriptContent content={data.script_contents} />
          <ScriptResult
            hostname={data.hostname}
            hostTimeout={data.host_timeout}
            exitCode={data.exit_code}
            message={data.message}
            output={data.output}
          />
        </>
      );
    }

    return (
      <>
        <div className={`${baseClass}__modal-content`}>{content}</div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    );
  };

  return (
    <Modal
      title="Script details"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {renderContent()}
    </Modal>
  );
};

export default ScriptDetailsModal;
