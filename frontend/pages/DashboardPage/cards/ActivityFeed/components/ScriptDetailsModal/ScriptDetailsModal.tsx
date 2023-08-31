import React from "react";
import { useQuery } from "react-query";

import scriptsAPI from "services/entities/scripts";

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
      <Icon name="pending-partial" />
      Script is running. To see if the script finished, close this modal and
      open it again.
    </p>
  </div>
);

const StatusMessageSuccess = () => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="success-partial" />
      {`Exit code: 0 (Script ran successfully.)`}
    </p>
  </div>
);

const StatusMessageFailed = ({ exitCode }: { exitCode: number }) => (
  <div className={`${baseClass}__status-message`}>
    <p>
      <Icon name="error-outline" />
      {`Exit code: ${exitCode} (Script failed.)`}
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
        <StatusMessageRunning />
      ) : (
        <StatusMessageError message={message} />
      );
    case -2:
      return <StatusMessageError message={message} />;
    case -1:
      return <StatusMessageError message={message} />;
    case 0:
      return <StatusMessageSuccess />;
    default:
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
  runtime: number;
}

const ScriptResult = ({
  hostname,
  hostTimeout,
  exitCode,
  message,
  output,
  runtime,
}: IScriptResultProps) => {
  const showOutputText = exitCode !== -998 && exitCode !== -999 && runtime < 30;

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
  onCancel: () => void;
}

const ScriptDetailsModal = ({ onCancel }: IScriptDetailsModalProps) => {
  // TODO: type for data
  const { data, isLoading, isError } = useQuery<any>(
    ["scriptDetailsModal"],
    () => {
      return scriptsAPI.getScriptResult(1);
    },
    { refetchOnWindowFocus: false }
  );

  const renderContent = () => {
    let content: JSX.Element;

    if (isLoading) {
      content = <Spinner />;
    } else if (isError) {
      content = <DataError description="Close this modal and try again." />;
    } else {
      content = (
        <>
          <ScriptContent content={data.script_contents} />
          <ScriptResult
            hostname={data.hostname}
            hostTimeout={data.host_timeout}
            exitCode={data.exit_code}
            message={data.message}
            output={data.output}
            runtime={data.runtime}
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
      title={"Script Details"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      {renderContent()}
    </Modal>
  );
};

export default ScriptDetailsModal;
