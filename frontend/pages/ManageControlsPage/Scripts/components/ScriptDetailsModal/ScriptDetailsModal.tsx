import React, { useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { format } from "date-fns";
import FileSaver from "file-saver";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import scriptAPI from "services/entities/scripts";
import { IHostScript } from "interfaces/script";
import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import Icon from "components/Icon";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import paths from "router/paths";
import ActionsDropdown from "components/ActionsDropdown";
import { generateActionDropdownOptions } from "pages/hosts/details/HostDetailsPage/modals/RunScriptModal/ScriptsTableConfig";

const baseClass = "script-details-modal";

interface IScriptDetailsModalProps {
  onCancel: () => void;
  onDelete: () => void;
  runScriptHelpText?: boolean;
  showHostScriptActions?: boolean;
  toggleRunScriptDetailsModal?: any;
  setRunScriptRequested?: (value: boolean) => void;
  hostId?: number | null;
  hostTeamId?: number | null;
  refetchHostScripts?: any;
  selectedScriptDetails: IHostScript;
}

const ScriptDetailsModal = ({
  onCancel,
  onDelete,
  runScriptHelpText = false,
  showHostScriptActions = false,
  toggleRunScriptDetailsModal,
  setRunScriptRequested,
  hostId,
  hostTeamId,
  refetchHostScripts,
  selectedScriptDetails,
}: IScriptDetailsModalProps) => {
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  // Note: Script metadata and script content require two separate API calls
  // Source: https://fleetdm.com/docs/rest-api/rest-api#example-get-script
  // So to get script name, we pass it into this modal instead of another API call
  // If in future iterations we want more script metadata, call scriptAPI.getScript()
  // and consider refactoring .getScript to return script content as well
  const {
    data: scriptContent,
    error: isScriptContentError,
    isLoading: isLoadingScriptContent,
  } = useQuery<any, Error>(
    ["scriptContent"],
    () => scriptAPI.downloadScript(selectedScriptDetails.script_id),
    {
      refetchOnWindowFocus: false,
    }
  );

  const onClickDownload = () => {
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const filename = `${formatDate} ${selectedScriptDetails}`;
    const file = new File([scriptContent], filename);
    FileSaver.saveAs(file);
  };

  const onSelectMoreActions = useCallback(
    async (action: string, script: IHostScript) => {
      if (hostId && setRunScriptRequested && refetchHostScripts) {
        switch (action) {
          case "showRunDetails": {
            toggleRunScriptDetailsModal(script);
            break;
          }
          case "run": {
            try {
              setRunScriptRequested && setRunScriptRequested(true);
              await scriptAPI.runScript({
                host_id: hostId,
                script_id: script.script_id,
              });
              renderFlash(
                "success",
                "Script is running or will run when the host comes online."
              );
              refetchHostScripts();
            } catch (e) {
              renderFlash("error", getErrorReason(e));
              setRunScriptRequested(false);
            }
            break;
          }
          default: // do nothing
        }
      }
    },
    [
      hostId,
      refetchHostScripts,
      renderFlash,
      setRunScriptRequested,
      toggleRunScriptDetailsModal,
    ]
  );

  const renderFooter = () => {
    if (isLoadingScriptContent) {
      return <></>;
    }
    return (
      <>
        <div className="modal-actions">
          <Button
            className={`${baseClass}__action-button`}
            variant="icon"
            onClick={() => onClickDownload()}
          >
            <Icon name="download" />
          </Button>
          <Button
            className={`${baseClass}__action-button`}
            variant="icon"
            onClick={onDelete}
          >
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        </div>{" "}
        <div className="modal-cta-wrap">
          {showHostScriptActions && (
            <div className={`${baseClass}__manage-automations-wrapper`}>
              <ActionsDropdown
                className={`${baseClass}__manage-automations-dropdown`}
                onChange={(value) =>
                  onSelectMoreActions(value, selectedScriptDetails)
                }
                placeholder="More actions"
                isSearchable={false}
                options={generateActionDropdownOptions(
                  currentUser,
                  hostTeamId || null,
                  selectedScriptDetails
                )}
              />
            </div>
          )}
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    );
  };

  const renderContent = () => {
    if (isLoadingScriptContent) {
      return <Spinner />;
    }

    if (isScriptContentError) {
      return <DataError description="Close this modal and try again." />;
    }

    return (
      <InputField
        readOnly
        inputWrapperClass={`${baseClass}__script-content`}
        name="script-content"
        label="Script content:"
        type="textarea"
        value={scriptContent}
        helpText={
          runScriptHelpText ? (
            <>
              To run this script on a host, go to the{" "}
              <CustomLink text="Hosts" url={paths.MANAGE_HOSTS} /> page and
              select a host.
              <br />
              To run the script across multiple hosts, add a policy automation
              on the <CustomLink
                text="Policies"
                url={paths.MANAGE_POLICIES}
              />{" "}
              page.
            </>
          ) : null
        }
        autoExpand
      />
    );
  };

  return (
    <Modal
      className={baseClass}
      title={selectedScriptDetails?.name || "Script details"}
      width="large"
      onExit={onCancel}
      modalActionsFooter={renderFooter()}
    >
      {renderContent()}
    </Modal>
  );
};

export default ScriptDetailsModal;
