import React, { useCallback, useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Editor from "components/Editor";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import paths from "router/paths";

import { getErrorMessage } from "../ScriptUploader/helpers";
import { ScriptContent } from "interfaces/script";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "edit-script-modal";

interface IEditScriptModal {
  onExit: () => void;
  scriptId: number;
  scriptName: string;
}

const EditScriptModal = ({
  scriptId,
  scriptName,
  onExit,
}: IEditScriptModal) => {
  const { renderFlash } = useContext(NotificationContext);

  // Editable script content
  // const [scriptFormData, setScriptFormData] = useState("");
  interface IEditScriptFormData {
    scriptContent: string;
  }
  const [scriptFormData, setScriptFormData] = useState<IEditScriptFormData>({
    scriptContent: "",
  });

  // TODO - validation, error states

  const {
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<ScriptContent, Error>(
    [scriptId],
    () => scriptAPI.downloadScript(scriptId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      onSuccess: (scriptContent) => {
        setScriptFormData({ scriptContent });
      },
    }
  );

  const [isSubmitting, setIsSubmitting] = useState(false);

  const onChange = (value: string) => {
    const newFormData = { ...scriptFormData, scriptContent: value };
    setScriptFormData(newFormData);
    setFormErrs(validate(newFormData));
  };

  const onSave = async () => {
    if (isSubmitting) {
      return;
    }
    try {
      setIsSubmitting(true);
      await scriptAPI.updateScript(scriptId, scriptFormData, scriptName);
      renderFlash("success", "Successfully saved script.");
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    } finally {
      setIsSubmitting(false);
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

    return (
      <>
        <form onSubmit={onSubmit}>
          <Editor
            value={scriptFormData.scriptContent}
            onChange={onChange}
            isFormField
            error={}
          />
          <div className="form-field__help-text">
            To run this script on a host, go to the{" "}
            <CustomLink text="Hosts" url={paths.MANAGE_HOSTS} /> page and select
            a host.
            <br />
            To run the script across multiple hosts, add a policy automation on
            the <CustomLink text="Policies" url={paths.MANAGE_POLICIES} /> page.
          </div>
        </form>
        <ModalFooter
          primaryButtons={
            <>
              <Button onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button onClick={onSave} variant="brand" isLoading={isSubmitting}>
                Save
              </Button>
            </>
          }
        />
      </>
    );
  };

  return (
    <Modal
      className={baseClass}
      title={scriptName}
      width="large"
      onExit={onExit}
    >
      {renderContent()}
    </Modal>
  );
};

export default EditScriptModal;
