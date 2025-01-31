import { format } from "date-fns";
import React, {
  useContext,
  useEffect,
  useState,
} from "react";
import {
  QueryObserverResult,
  RefetchOptions,
  RefetchQueryFilters,
  useQuery,
} from "react-query";

import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import scriptAPI, { IHostScriptsResponse } from "services/entities/scripts";


import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Editor from "components/Editor";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import paths from "router/paths";

import { getErrorMessage } from "../ScriptUploader/helpers";

const baseClass = "edit-script-modal";

interface IEditScriptModal {
  onCancel: () => void;
  scriptId: number;
  scriptName: string;
  isHidden?: boolean;
  refetchHostScripts?: <TPageData>(
    options?: (RefetchOptions & RefetchQueryFilters<TPageData>) | undefined,
  ) => Promise<QueryObserverResult<IHostScriptsResponse, IApiError>>;
}

const EditScriptModal = ({ scriptId, scriptName, onCancel, isHidden, refetchHostScripts }: IEditScriptModal) => {
  const { renderFlash } = useContext(NotificationContext);

  const {
    data: scriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<any, Error>(
    [scriptId],
    () => scriptAPI.downloadScript(scriptId),
    {
      refetchOnWindowFocus: false,
    },
  );

  useEffect(() => {
    if (isSelectedScriptContentError != null) {

    }
  }, [isSelectedScriptContentError])

  // Editable script content
  const [scriptFormData, setScriptFormData] = useState("");
  useEffect(() => {
    setScriptFormData(scriptContent)
  }, [scriptContent]);

  const handleOnChange = (value: string) => {
    setScriptFormData(value);
  }

  const onUpload = () => {
    onCancel();
  }

  const handleOnSave = async () => {
    try {
      await scriptAPI.updateScript(scriptId, scriptFormData, scriptName)
      renderFlash("success", "Successfully saved script.");
      onUpload();
    } catch (e) {
      renderFlash("error", getErrorMessage(e) );
    }
  }

  const renderContent = () => {
    if (isLoadingSelectedScriptContent) {
      return <Spinner />;
    }

    if (isSelectedScriptContentError) {
      return <DataError description="Close this modal and try again." />;
    }

    return (
      <>
        <form>
          <Editor
            value={scriptFormData}
            onChange={handleOnChange}
            isFormField={true}
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
              <Button onClick={onCancel} variant="inverse">
                Cancel
              </Button>
              <Button onClick={handleOnSave} variant="brand">
                Save
              </Button>
            </>
          }
        />
      </>
    )
  };

  return (
    <Modal
      className={baseClass}
      title={scriptName}
      width="large"
      onExit={onCancel}
      isHidden={isHidden}
    >
      {renderContent()}
    </Modal>
  )
};

export default EditScriptModal;
