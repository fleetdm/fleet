/** controls/scripts/library/:id */

import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import { SingleValue } from "react-select-5";

import paths from "router/paths";
import scriptAPI, { IScriptResponse } from "services/entities/scripts";
import { IScript, ScriptContent } from "interfaces/script";
import { ignoreAxiosError } from "interfaces/errors";
import {
  APP_CONTEXT_NO_TEAM_ID,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { internationalTimeFormat } from "utilities/helpers";
import { getPathWithQueryParams } from "utilities/url";

import useGitOpsMode from "hooks/useGitOpsMode";

import BackButton from "components/BackButton";
import Card from "components/Card";
import DataError from "components/DataError";
import DataSet from "components/DataSet";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import Editor from "components/Editor";
import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";
import MainContent from "components/MainContent";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TooltipTruncatedText from "components/TooltipTruncatedText";

import RunScriptHelpText from "pages/hosts/components/ScriptDetailsModal/RunScriptHelpText";

import DeleteScriptModal from "../components/DeleteScriptModal";
import { getErrorMessage as getUploadErrorMessage } from "../components/ScriptUploadModal/helpers";

const baseClass = "script-details-page";

const ACTION_SAVE = "save";
const ACTION_DELETE = "delete";

interface ISaveWarningModalProps {
  scriptName: string;
  isSubmitting: boolean;
  onExit: () => void;
  onSave: () => void;
}

const SaveWarningModal = ({
  scriptName,
  isSubmitting,
  onExit,
  onSave,
}: ISaveWarningModalProps) => (
  <Modal title="Save changes?" onExit={onExit}>
    <>
      <p>
        The changes you are making will cancel any pending script runs for{" "}
        <b>{scriptName}</b>.<br />
        <br />
        If this script is currently running on a host, it will complete, but
        results won&apos;t appear in Fleet.
        <br />
        <br />
        You cannot undo this action.
      </p>
      <div className="modal-cta-wrap">
        <Button onClick={onSave} isLoading={isSubmitting}>
          Save
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </>
  </Modal>
);

const validate = (scriptContent: string) =>
  scriptContent.trim() === "" ? "Script cannot be empty" : null;

const getEditorMode = (scriptName: string) => {
  if (scriptName.match(/\.ps1$/)) return "powershell";
  if (scriptName.match(/\.py$/)) return "python";
  return "sh";
};

const getScriptGraphic = (scriptName: string): GraphicNames => {
  const ext = scriptName.split(".").pop();
  switch (ext) {
    case "py":
      return "file-py";
    case "sh":
      return "file-sh";
    case "ps1":
      return "file-ps1";
    default:
      return "file-script";
  }
};

interface IScriptDetailsRouteParams {
  id: string;
}

type IScriptDetailsPageProps = RouteComponentProps<
  undefined,
  IScriptDetailsRouteParams
>;

const ScriptDetailsPage = ({
  router,
  routeParams,
}: IScriptDetailsPageProps) => {
  const scriptId = parseInt(routeParams.id, 10);
  const handlePageError = useErrorHandler();
  const { renderFlash } = useContext(NotificationContext);
  const {
    availableTeams,
    currentUser,
    isGlobalAdmin,
    isAnyTeamAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainer,
    isGlobalTechnician,
    isTeamTechnician,
  } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();

  const isTechnician = !!isGlobalTechnician || !!isTeamTechnician;
  const canRunScripts = !!(
    isGlobalAdmin ||
    isAnyTeamAdmin ||
    isGlobalMaintainer ||
    isAnyTeamMaintainer
  );
  const canManageScripts = !isTechnician && !gitOpsModeEnabled;

  const [scriptFormData, setScriptFormData] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [showSaveWarning, setShowSaveWarning] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const {
    data: script,
    isLoading: isLoadingScript,
    isError: isScriptError,
  } = useQuery<IScriptResponse, AxiosError, IScript>(
    ["script", scriptId],
    () => scriptAPI.getScript(scriptId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !Number.isNaN(scriptId),
      onError: (error) => {
        if (!ignoreAxiosError(error, [403, 404])) {
          handlePageError(error);
        }
      },
    }
  );

  const {
    data: scriptContent,
    isLoading: isLoadingContent,
    isError: isContentError,
  } = useQuery<ScriptContent, AxiosError>(
    ["script-content", scriptId],
    () => scriptAPI.downloadScript(scriptId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !Number.isNaN(scriptId),
      onSuccess: (content) => {
        setScriptFormData(content);
      },
    }
  );

  const onChange = (value: string) => {
    setScriptFormData(value);
    if (formError) {
      setFormError(validate(value));
    }
  };

  const onBlur = () => {
    setFormError(validate(scriptFormData));
  };

  const performSave = async () => {
    if (!script) return;
    try {
      setIsSubmitting(true);
      await scriptAPI.updateScript(scriptId, scriptFormData, script.name);
      renderFlash("success", "Successfully saved script.");
      setShowSaveWarning(false);
    } catch (e) {
      renderFlash("error", getUploadErrorMessage(e));
    } finally {
      setIsSubmitting(false);
    }
  };

  const onClickSave = () => {
    const err = validate(scriptFormData);
    setFormError(err);
    if (err || isSubmitting) return;
    if (scriptContent !== scriptFormData) {
      setShowSaveWarning(true);
      return;
    }
    performSave();
  };

  const backPath = getPathWithQueryParams(paths.CONTROLS_SCRIPTS_LIBRARY, {
    fleet_id: script?.team_id ?? APP_CONTEXT_NO_TEAM_ID,
  });

  const onAfterDelete = () => {
    setShowDeleteModal(false);
    router.push(backPath);
  };

  const teamLabel = (() => {
    if (script?.team_id == null) return APP_CONTEXT_NO_TEAM_SUMMARY.name;
    if (script.team_id === APP_CONTEXT_NO_TEAM_ID) {
      return APP_CONTEXT_NO_TEAM_SUMMARY.name;
    }
    const team = availableTeams?.find((t) => t.id === script.team_id);
    return team?.name ?? `Fleet ${script.team_id}`;
  })();

  const isDirty =
    scriptContent !== undefined && scriptContent !== scriptFormData;
  const canSave = canManageScripts && isDirty && !formError && !isSubmitting;

  const actionOptions: CustomOptionType[] = [
    {
      label: "Save",
      value: ACTION_SAVE,
      isDisabled: !canSave,
      tooltipContent: gitOpsModeEnabled
        ? "Editing is disabled in GitOps mode."
        : undefined,
    },
    {
      label: "Delete",
      value: ACTION_DELETE,
      isDisabled: !canManageScripts,
    },
  ];

  const onSelectAction = (option: SingleValue<CustomOptionType>) => {
    switch (option?.value) {
      case ACTION_SAVE:
        onClickSave();
        break;
      case ACTION_DELETE:
        setShowDeleteModal(true);
        break;
      default:
    }
  };

  const renderContent = () => {
    if (Number.isNaN(scriptId)) {
      return <DataError description="Invalid script ID." />;
    }
    if (isLoadingScript || isLoadingContent) {
      return <Spinner />;
    }
    if (isScriptError || isContentError || !script) {
      return <DataError description="Could not load script." />;
    }

    return (
      <Card borderRadiusSize="xxlarge" className={`${baseClass}__summary-card`}>
        <div className={`${baseClass}__summary`}>
          <Graphic
            className={`${baseClass}__graphic`}
            name={getScriptGraphic(script.name)}
          />
          <div className={`${baseClass}__info`}>
            <div className={`${baseClass}__title-actions`}>
              <h1 className={`${baseClass}__title`}>
                <TooltipTruncatedText value={script.name} />
              </h1>
              {currentUser && (
                <div className={`${baseClass}__actions-dropdown`}>
                  <DropdownWrapper
                    name="script-actions"
                    placeholder="Actions"
                    options={actionOptions}
                    onChange={onSelectAction}
                    variant="button"
                    nowrapMenu
                  />
                </div>
              )}
            </div>
            <dl className={`${baseClass}__description-list`}>
              <DataSet title="Fleet" value={teamLabel} />
              <DataSet
                title="Uploaded"
                value={internationalTimeFormat(new Date(script.created_at))}
              />
              <DataSet
                title="Last modified"
                value={internationalTimeFormat(new Date(script.updated_at))}
              />
            </dl>
          </div>
        </div>
        <div className={`${baseClass}__editor`}>
          <Editor
            mode={getEditorMode(script.name)}
            label="Script"
            error={formError}
            value={scriptFormData}
            onBlur={onBlur}
            onChange={onChange}
            readOnly={!canManageScripts}
            maxLines={30}
          />
          <RunScriptHelpText
            className="form-field__help-text"
            isTechnician={isTechnician}
            canRunScripts={canRunScripts}
            teamId={script.team_id ?? undefined}
          />
        </div>
      </Card>
    );
  };

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}__header-links`}>
          <BackButton text="Back to scripts" path={backPath} />
        </div>
        {renderContent()}
      </MainContent>
      {showSaveWarning && script && (
        <SaveWarningModal
          scriptName={script.name}
          isSubmitting={isSubmitting}
          onExit={() => setShowSaveWarning(false)}
          onSave={performSave}
        />
      )}
      {showDeleteModal && script && (
        <DeleteScriptModal
          scriptName={script.name}
          scriptId={script.id}
          onCancel={() => setShowDeleteModal(false)}
          afterDelete={onAfterDelete}
        />
      )}
    </>
  );
};

export default ScriptDetailsPage;
