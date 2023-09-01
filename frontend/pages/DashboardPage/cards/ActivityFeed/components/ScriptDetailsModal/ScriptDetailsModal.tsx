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
  let statusMessage: JSX.Element;

  const fleetStoppedScript = exitCode === -1;
  const scriptsDisabledForHost = exitCode === -2;
  const hostTimedOut = exitCode === null && hostTimeout === true;
  const scriptStillRunning = exitCode === null && hostTimeout === false;

  if (exitCode === 0) {
    statusMessage = (
      <p>
        <Icon name="success-partial" />
        Exit code: 0 (Script ran successfully.)
      </p>
    );
  } else if (fleetStoppedScript || scriptsDisabledForHost || hostTimedOut) {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        {`Error: ${message}`}
      </p>
    );
  } else if (scriptStillRunning) {
    statusMessage = (
      <p>
        <Icon name="pending-partial" />
        {message}
      </p>
    );
  } else {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        {`Exit code: ${exitCode}: (Script failed.)`}
      </p>
    );
  }

  return <div className={`${baseClass}__status-message`}>{statusMessage}</div>;
};

interface IScriptOutputProps {
  output: string;
}

const ScriptOutput = ({ output }: IScriptOutputProps) => {
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
        when <b>Marko&apos;s MacBook Pro</b> ran the script above:
      </p>
      <Textarea className={`${baseClass}__output-textarea`}>{output}</Textarea>
    </div>
  );
};

interface IScriptResultProps {
  hostTimeout: boolean;
  exitCode: number | null;
  message: string;
  output: string;
}

const ScriptResult = ({
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
      {showOutputText && <ScriptOutput output={output} />}
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
