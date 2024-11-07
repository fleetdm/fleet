import React, { useCallback, useContext } from "react";
import { format } from "date-fns";
import { useQuery } from "react-query";
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
import Textarea from "components/Textarea";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import paths from "router/paths";
import ActionsDropdown from "components/ActionsDropdown";
import { generateActionDropdownOptions } from "pages/hosts/details/HostDetailsPage/modals/RunScriptModal/ScriptsTableConfig";

const baseClass = "script-details-modal";

type PartialOrFullHostScript =
  | Pick<IHostScript, "script_id" | "name"> // Use on Scripts page does not include last_execution
  | IHostScript;

interface IScriptDetailsModalProps {
  onCancel: () => void;
  onDelete: () => void;
  runScriptHelpText?: boolean;
  showHostScriptActions?: boolean;
  setRunScriptRequested?: (value: boolean) => void;
  hostId?: number | null;
  hostTeamId?: number | null;
  refetchHostScripts?: any;
  selectedScriptDetails?: PartialOrFullHostScript;
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

  const {
    data: scriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<any, Error>(
    ["scriptContent", selectedScriptDetails?.script_id],
    () =>
      selectedScriptDetails
        ? scriptAPI.downloadScript(selectedScriptDetails.script_id!)
        : Promise.resolve(null),
    {
      refetchOnWindowFocus: false,
      enabled: !selectedScriptContent && !!selectedScriptDetails?.script_id,
    }
  );
  const getScriptContent = async () => {
    try {
      const content = selectedScriptContent || scriptContent;
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
      const filename = `${formatDate} ${selectedScriptDetails?.name}`;
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

  const renderFooter = () => {
    if (!shouldShowFooter) {
      return <></>;
    }

    return (
      <>
        <div className={`secondary-actions ${baseClass}__script-actions`}>
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
        </div>
        <div className={`primary-actions ${baseClass}__host-script-actions`}>
          {showHostScriptActions && selectedScriptDetails && (
            <div className={`${baseClass}__manage-automations-wrapper`}>
              <ActionsDropdown
                className={`${baseClass}__manage-automations-dropdown`}
                onChange={(value) =>
                  onSelectMoreActions(
                    value,
                    selectedScriptDetails as IHostScript
                  )
                }
                placeholder="More actions"
                isSearchable={false}
                options={generateActionDropdownOptions(
                  currentUser,
                  hostTeamId || null,
                  selectedScriptDetails as IHostScript
                )}
                menuPlacement="top"
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
    if (isLoadingScriptContent || isLoadingSelectedScriptContent) {
      return <Spinner />;
    }

    if (isScriptContentError || isSelectedScriptContentError) {
      return <DataError description="Close this modal and try again." />;
    }

    return (
      <div className={`${baseClass}__script-content`}>
        <span>Script content:</span>
        <Textarea className={`${baseClass}__script-content-textarea`}>
          {scriptContent}
        </Textarea>
        {runScriptHelpText && (
          <div className="form-field__help-text">
            To run this script on a host, go to the{" "}
            <CustomLink text="Hosts" url={paths.MANAGE_HOSTS} /> page and select
            a host.
            <br />
            To run the script across multiple hosts, add a policy automation on
            the <CustomLink text="Policies" url={paths.MANAGE_POLICIES} /> page.
          </div>
        )}
      </div>
    );
  };

  return (
    <Modal
      className={baseClass}
      title={selectedScriptDetails?.name || "Script details"}
      width="large"
      onExit={onCancel}
      actionsFooter={shouldShowFooter() ? renderFooter() : undefined}
      isHidden={isHidden}
    >
      {renderContent()}
    </Modal>
  );
};

export default ScriptDetailsModal;
