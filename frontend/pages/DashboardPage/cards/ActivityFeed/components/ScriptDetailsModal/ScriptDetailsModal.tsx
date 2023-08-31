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

const SCRIPT_RUNNING_CODE = -999;
const HOST_NOT_REACHED_CODE = -998;

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
  runtime: number;
}

const StatusMessage = ({
  hostTimeout,
  exitCode,
  message,
  runtime,
}: IStatusMessageProps) => {
  let statusMessage: JSX.Element;

  const scriptStillRunning = exitCode === null && hostTimeout === false;
  const noHostResponse = exitCode === null && hostTimeout === true;
  const scriptsAreDisabledForHost = exitCode === -2 && hostTimeout === true;

  // script timed out error
  if (hostTimeout) {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        Error: Timeout. Fleet stopped the script after 30 seconds to protect
        host performance.
      </p>
    );
    // host could not be reached
    // TODO: clarify what causes this message to show.
  } else if (noHostResponse) {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        Error: Fleet hasn&apos;t heard from the host in over 1 minute because it
        went offline. Run the script again when the host comes back online.
      </p>
    );
  } else if (scriptsAreDisabledForHost) {
    statusMessage = (
      <p>
        <Icon name="error-outline" />
        Error: Scripts are disabled for this host. To run scripts, deploy a
        Fleet installer with scripts enabled.
      </p>
    );
  } else if (scriptStillRunning) {
    statusMessage = (
      <p>
        <Icon name="pending-partial" />
        Script is running. To see if the script finished, close this modal and
        open it again.
      </p>
    );
  } else {
    // 0 or 1 exit code with message
    statusMessage = (
      <p>
        <Icon name={exitCode === 0 ? "success-partial" : "error-outline"} />
        {`Exit code: ${exitCode} (${message})`}
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
        runtime={runtime}
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
