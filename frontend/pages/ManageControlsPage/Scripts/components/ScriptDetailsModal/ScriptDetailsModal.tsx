import React, { useContext } from "react";
import { useQuery } from "react-query";

import scriptAPI from "services/entities/scripts";
import { NotificationContext } from "context/notification";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import paths from "router/paths";
import { AxiosResponse } from "axios";
import { IApiError } from "../../../../../interfaces/errors";
import { getErrorMessage } from "../ScriptUploader/helpers";

const baseClass = "script-details-modal";

interface IScriptDetailsModalProps {
  scriptName: string;
  scriptId: number;
  onCancel: () => void;
  onDownload: () => void;
  onDelete: (script: string) => void;
}

const ScriptDetailsModal = ({
  scriptName,
  scriptId,
  onCancel,
  onDownload,
  onDelete,
}: IScriptDetailsModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const {
    data: script,
    error: fetchScriptError,
    isLoading: isLoadingScript,
    isFetching: isFetchingScript,
  } = useQuery<string, Error>(
    ["certificate"],
    () => scriptAPI.downloadScript(scriptId),
    {
      refetchOnWindowFocus: false,
    }
  );

  return (
    <Modal
      className={baseClass}
      title={scriptName}
      onExit={onCancel}
      onEnter={onCancel}
      modalActionsFooter={
        <>
          <div className="modal-actions">
            <Button
              className={`${baseClass}__action-button`}
              variant="text-icon"
              onClick={onDownload}
            >
              <Icon name="download" />
            </Button>
            <Button
              className={`${baseClass}__action-button`}
              variant="text-icon"
              onClick={() => onDelete(script)}
            >
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          </div>{" "}
          <div className="modal-cta-wrap">
            <Button onClick={onCancel} variant="brand">
              Done
            </Button>
          </div>
        </>
      }
    >
      <>
        {isLoadingScript ? (
          <Spinner />
        ) : (
          <InputField
            readOnly
            inputWrapperClass={`${baseClass}__script-content`}
            name="script-content"
            label="Script content:"
            type="textarea"
            value={script}
            helpText={
              <>
                To run this script on a host, go to the{" "}
                <CustomLink text="Hosts" url={paths.MANAGE_HOSTS} /> page and
                select a host.
                <br />
                To run the script across multiple hosts, add a policy automation
                on the{" "}
                <CustomLink text="Policies" url={paths.MANAGE_POLICIES} /> page.
              </>
            }
          />
        )}
      </>
    </Modal>
  );
};

export default ScriptDetailsModal;
