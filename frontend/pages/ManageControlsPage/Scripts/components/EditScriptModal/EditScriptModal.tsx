import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import classnames from "classnames";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { getPathWithQueryParams } from "utilities/url";
import scriptAPI from "services/entities/scripts";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Editor from "components/Editor";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import paths from "router/paths";

import { ScriptContent } from "interfaces/script";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getErrorMessage } from "../ScriptUploader/helpers";

const baseClass = "edit-script-modal";

interface IWarningModal {
  onExit: () => void;
  onSave: () => void;
  scriptName: string;
  isSubmitting: boolean;
}
const WarningModal = ({
  onExit,
  onSave,
  scriptName,
  isSubmitting,
}: IWarningModal) => {
  return (
    <Modal
      className={`${baseClass}__warning`}
      title="Save changes?"
      onExit={onExit}
    >
      <>
        <p>
          The changes you are making will cancel any pending script runs for{" "}
          <b>{scriptName}</b>.<br />
          <br />
          If this script is currently running on a host, it will complete, but
          results won&apos;t appear in Fleet. <br />
          <br />
          You cannot undo this action.
        </p>
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSave}
            className="save-loading"
            isLoading={isSubmitting}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

interface IEditScriptModal {
  onExit: () => void;
  scriptId: number;
  scriptName: string;
}

const validate = (scriptContent: string) => {
  if (scriptContent.trim() === "") {
    return "Script cannot be empty";
  }
  return null;
};

const EditScriptModal = ({
  scriptId,
  scriptName,
  onExit,
}: IEditScriptModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const { currentTeam } = useContext(AppContext);

  // Editable script content
  const [scriptFormData, setScriptFormData] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [showConfirmChanges, setShowConfirmChanges] = useState(false);

  const {
    data: curScriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<ScriptContent, Error>(
    [scriptId],
    () => scriptAPI.downloadScript(scriptId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      onSuccess: (curScriptContent_) => {
        setScriptFormData(curScriptContent_);
      },
    }
  );

  const onChange = (value: string) => {
    setScriptFormData(value);
    const err = validate(value);
    if (!err && !!formError) {
      setFormError(validate(value));
    }
  };

  const onBlur = () => {
    setFormError(validate(scriptFormData));
  };

  const onSave = async () => {
    const err = validate(scriptFormData);
    setFormError(err);
    if (err || isSubmitting) {
      return;
    }
    // if contents have changed and not already showing the warning modal
    if (curScriptContent !== scriptFormData && !showConfirmChanges) {
      setShowConfirmChanges(true);
      return;
    }
    // show warning modal and abort this call
    try {
      setIsSubmitting(true);
      await scriptAPI.updateScript(scriptId, scriptFormData, scriptName);
      renderFlash("success", "Successfully saved script.");
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    } finally {
      setIsSubmitting(false);
      setShowConfirmChanges(false);
    }
  };

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onSave();
  };

  const renderContent = () => {
    if (isLoadingSelectedScriptContent) {
      return <Spinner />;
    }

    if (isSelectedScriptContentError) {
      return <DataError description="Close this modal and try again." />;
    }

    // Set editing mode based on the file extension.
    const mode = scriptName.match(/\.sh$/) ? "sh" : "powershell";

    return (
      <>
        <form onSubmit={onSubmit}>
          <Editor
            mode={mode}
            error={formError}
            label="Script"
            onBlur={onBlur}
            onChange={onChange}
            value={scriptFormData}
          />
          <div className="form-field__help-text">
            To run this script on a host, go to the{" "}
            <CustomLink
              text="Hosts"
              url={getPathWithQueryParams(paths.MANAGE_HOSTS, {
                team_id: currentTeam?.id,
              })}
            />{" "}
            page and select a host.
            <br />
            To run the script across multiple hosts, add a policy automation on
            the{" "}
            <CustomLink
              text="Policies"
              url={getPathWithQueryParams(paths.MANAGE_POLICIES, {
                team_id: currentTeam?.id,
              })}
            />{" "}
            page.
          </div>
        </form>
        <ModalFooter
          primaryButtons={
            <>
              <Button onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button
                onClick={onSave}
                isLoading={isSubmitting}
                disabled={!!formError}
              >
                Save
              </Button>
            </>
          }
        />
      </>
    );
  };

  const classes = classnames(baseClass, {
    [`${baseClass}__hide-main`]: !!showConfirmChanges,
  });
  return (
    <>
      <Modal
        className={classes}
        title={scriptName}
        width="large"
        onExit={onExit}
      >
        {renderContent()}
      </Modal>
      {!!showConfirmChanges && (
        <WarningModal
          onExit={() => setShowConfirmChanges(false)}
          onSave={onSave}
          scriptName={scriptName}
          isSubmitting={isSubmitting}
        />
      )}
    </>
  );
};

export default EditScriptModal;
