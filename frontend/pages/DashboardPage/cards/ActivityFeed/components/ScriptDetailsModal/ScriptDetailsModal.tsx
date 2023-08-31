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

  const scriptStillRunning = exitCode === null && hostTimeout === false;
  const hasExitCode = exitCode !== null;

  // The messaging to the user is hardcoded when the script is still running
  // OR when the script has an exit code is not null.
  // Whenever the exit code is null we display the message that the API sends back.
  if (scriptStillRunning && !message) {
    statusMessage = (
      <p>
        <Icon name="pending-partial" />
        Script is running. To see if the script finished, close this modal and
        open it again.
      </p>
    );
  } else if (hasExitCode) {
    // 0 or 1 exit code with message
    const exitCodeMessage =
      exitCode === 0 ? "Script ran successfully" : "Script failed";
    statusMessage = (
      <p>
        <Icon name={exitCode === 0 ? "success-partial" : "error-outline"} />
        {`Exit code: ${exitCode} (${exitCodeMessage}.)`}
      </p>
    );
  } else {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        {message}
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
  runtime: number;
}

const ScriptResult = ({
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
