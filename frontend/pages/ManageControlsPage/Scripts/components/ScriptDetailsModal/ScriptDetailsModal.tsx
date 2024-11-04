import React, { useCallback, useContext } from "react";
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
  setRunScriptRequested?: (value: boolean) => void;
  hostId?: number | null;
  hostTeamId?: number | null;
  refetchHostScripts?: any;
  selectedScriptDetails?: IHostScript;
  selectedScriptContent?: string;
  isLoadingScriptContent?: boolean;
  isScriptContentError?: Error | null;
  isHidden?: boolean;
  onCloseScriptModalGroup?: () => void;
  onClickRunDetails?: (scriptExecutionId: string) => void;
}

const ScriptDetailsModal = ({
  onCancel,
  onDelete,
  runScriptHelpText = false,
  showHostScriptActions = false,
  setRunScriptRequested,
  hostId,
  hostTeamId,
  refetchHostScripts,
  selectedScriptDetails,
  selectedScriptContent,
  isLoadingScriptContent,
  isScriptContentError,
  isHidden = false,
  onCloseScriptModalGroup,
  onClickRunDetails,
}: IScriptDetailsModalProps) => {
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const getScriptContent = async () => {
    try {
      const content = await scriptAPI.downloadScript(
        selectedScriptDetails?.script_id || 1
      );
      const formatDate = format(new Date(), "yyyy-MM-dd");
      const filename = `${formatDate} ${
        selectedScriptDetails?.name || "Script details"
      }`;
      const file = new File([content], filename);
      FileSaver.saveAs(file);
    } catch {
      renderFlash("error", "Couldn’t Download. Please try again.");
    }
  };

  const onClickDownload = () => {
    if (selectedScriptContent) {
      const formatDate = format(new Date(), "yyyy-MM-dd");
      const filename = `${formatDate} ${selectedScriptDetails}`;
      const file = new File([selectedScriptContent], filename);
      FileSaver.saveAs(file);
    } else {
      getScriptContent();
    }
  };

  const onSelectMoreActions = useCallback(
    async (action: string, script: IHostScript) => {
      if (
        hostId &&
        setRunScriptRequested &&
        refetchHostScripts &&
        script.last_execution?.execution_id
      ) {
        switch (action) {
          case "showRunDetails": {
            onClickRunDetails &&
              onClickRunDetails(script.last_execution?.execution_id);
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
              onCloseScriptModalGroup && onCloseScriptModalGroup(); // Running a script closes the modal groups
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
      onClickRunDetails,
      setRunScriptRequested,
      refetchHostScripts,
      onCloseScriptModalGroup,
      renderFlash,
    ]
  );

  const shouldShowFooter = () => {
    return !isLoadingScriptContent && selectedScriptDetails !== undefined;
  };

  console.log("selectedScriptDetails", selectedScriptDetails);

  const renderFooter = () => {
    if (!shouldShowFooter) {
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
          {showHostScriptActions && selectedScriptDetails && (
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
        value={selectedScriptContent}
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

  console.log("shouldshowfooter", shouldShowFooter());
  return (
    <Modal
      className={baseClass}
      title={selectedScriptDetails?.name || "Script details"}
      width="large"
      onExit={onCancel}
      modalActionsFooter={shouldShowFooter() ? renderFooter() : undefined}
      isHidden={isHidden}
    >
      {renderContent()}
    </Modal>
  );
};

export default ScriptDetailsModal;
