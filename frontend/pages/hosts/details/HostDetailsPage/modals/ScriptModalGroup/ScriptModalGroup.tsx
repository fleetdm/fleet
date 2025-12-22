import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";

import { getErrorReason, IApiError } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { IHostScript } from "interfaces/script";
import { IUser } from "interfaces/user";

import scriptsAPI, {
  IHostScriptsQueryKey,
  IHostScriptsResponse,
} from "services/entities/scripts";

import { NotificationContext } from "context/notification";

import ScriptDetailsModal from "pages/hosts/components/ScriptDetailsModal";
import DeleteScriptModal from "pages/ManageControlsPage/Scripts/components/DeleteScriptModal";
import RunScriptDetailsModal from "pages/DashboardPage/cards/ActivityFeed/components/RunScriptDetailsModal";
import RunScriptModal from "../RunScriptModal";
import ConfirmRunScriptModal from "../ConfirmRunScriptModal";

interface IScriptsProps {
  currentUser: IUser | null;
  host: IHost;
  onCloseScriptModalGroup: () => void;
  teamIdForApi?: number;
}

enum ModalGroupOption {
  Run = "run",
  ConfirmRun = "confirm-run",
  ViewScriptDetails = "script-details",
  ViewRunDetails = "run-details",
  Delete = "delete",
}

const ScriptModalGroup = ({
  currentUser,
  host,
  onCloseScriptModalGroup,
  teamIdForApi,
}: IScriptsProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [previousModal, setPreviousModal] = useState<ModalGroupOption | null>(
    null
  );
  // this should never actually be null - nullability satisfies TS when setting current modal to previous
  const [currentModal, setCurrentModal] = useState<ModalGroupOption | null>(
    ModalGroupOption.Run
  );
  const [runScriptTablePage, setRunScriptTablePage] = useState(0);
  const [selectedExecutionId, setSelectedExecutionId] = useState<
    string | undefined
  >(undefined);

  const [selectedScript, setSelectedScript] = useState<IHostScript | null>(
    null
  );
  const [isRunningScript, setIsRunningScript] = useState(false);

  // Almost everything from this is needed on RunScript.tsx modal
  // except refetch is used multiple places
  const {
    data: runScriptTableResponse,
    isError: isErrorHostScripts,
    isLoading: isLoadingHostScripts,
    isFetching: isFetchingHostScripts,
    refetch: refetchHostScripts,
  } = useQuery<
    IHostScriptsResponse,
    IApiError,
    IHostScriptsResponse,
    IHostScriptsQueryKey[]
  >(
    [
      {
        scope: "host_scripts",
        host_id: host.id,
        page: runScriptTablePage,
        per_page: 10,
      },
    ],
    ({ queryKey }) => scriptsAPI.getHostScripts(queryKey[0]),
    {
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: 3000,
    }
  );

  // Note: Script metadata and script content require two separate API calls
  // Source: https://fleetdm.com/docs/rest-api/rest-api#example-get-script
  // So to get script name, we pass it into this modal instead of another API call
  // If in future iterations we want more script metadata, call scriptAPI.getScript()
  // and consider refactoring .getScript to return script content as well

  // TODO- move into script details modal, depend on selectedScript and modal being selected
  const {
    data: selectedScriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<string, Error>(
    ["scriptContent", selectedScript?.script_id],
    () => scriptsAPI.downloadScript(selectedScript?.script_id ?? 1),
    {
      refetchOnWindowFocus: false,
      enabled:
        !!selectedScript && currentModal === ModalGroupOption.ViewScriptDetails,
    }
  );

  const goBack = useCallback(() => {
    setCurrentModal(previousModal);
    setPreviousModal(null);
  }, [previousModal]);

  const onConfirmRunScript = useCallback(async () => {
    // will always be truthy at this point
    if (selectedScript) {
      try {
        setIsRunningScript(true);
        await scriptsAPI.runScript({
          host_id: host.id,
          // will be defined when this is being called
          script_id: selectedScript.script_id,
        });
        renderFlash(
          "success",
          "Script is running or will run when the host comes online."
        );
        refetchHostScripts();
      } catch (e) {
        renderFlash("error", getErrorReason(e));
      } finally {
        setIsRunningScript(false);
        setSelectedScript(null);
        setCurrentModal(ModalGroupOption.Run);
      }
    }
  }, [host.id, refetchHostScripts, renderFlash, selectedScript]);

  const onClikRunBeforeConfirmation = useCallback(
    (script: IHostScript) => {
      setPreviousModal(currentModal);
      setCurrentModal(ModalGroupOption.ConfirmRun);
      setSelectedScript(script);
    },
    [currentModal]
  );

  return (
    <>
      <RunScriptModal
        currentUser={currentUser}
        hostTeamId={host.team_id}
        onClickRun={onClikRunBeforeConfirmation}
        onClose={onCloseScriptModalGroup}
        onClickViewScript={(script: IHostScript) => {
          setPreviousModal(currentModal);
          setCurrentModal(ModalGroupOption.ViewScriptDetails);
          setSelectedScript(script);
        }}
        onClickRunDetails={(scriptExecutionId: string) => {
          setPreviousModal(currentModal);
          setCurrentModal(ModalGroupOption.ViewRunDetails);
          setSelectedExecutionId(scriptExecutionId);
        }}
        page={runScriptTablePage}
        setPage={setRunScriptTablePage}
        hostScriptResponse={runScriptTableResponse}
        isRunningScript={isRunningScript}
        isFetchingHostScripts={isFetchingHostScripts}
        isLoadingHostScripts={isLoadingHostScripts}
        isError={isErrorHostScripts}
        isHidden={currentModal !== ModalGroupOption.Run}
      />
      <ConfirmRunScriptModal
        onClose={onCloseScriptModalGroup}
        onCancel={() => {
          if (previousModal === ModalGroupOption.Run) {
            setSelectedScript(null);
          }
          goBack();
        }}
        onConfirmRunScript={onConfirmRunScript}
        scriptName={selectedScript?.name}
        hostName={host.display_name}
        isRunningScript={isRunningScript}
        isHidden={currentModal !== ModalGroupOption.ConfirmRun}
      />
      <ScriptDetailsModal
        hostTeamId={host.team_id}
        selectedScriptDetails={selectedScript}
        // script id and details both passed to accomodate various instances of this component, some
        // in a slightly different context
        selectedScriptId={selectedScript?.script_id}
        selectedScriptContent={selectedScriptContent}
        onClose={onCloseScriptModalGroup}
        onCancel={() => {
          setSelectedScript(null);
          goBack();
        }}
        onDelete={() => {
          setPreviousModal(currentModal);
          setCurrentModal(ModalGroupOption.Delete);
        }}
        onClickRunDetails={(scriptExecutionId: string) => {
          setPreviousModal(currentModal);
          setCurrentModal(ModalGroupOption.ViewRunDetails);
          scriptExecutionId && setSelectedExecutionId(scriptExecutionId);
        }}
        onClickRun={onClikRunBeforeConfirmation}
        isLoadingScriptContent={isLoadingSelectedScriptContent}
        isScriptContentError={isSelectedScriptContentError}
        isHidden={currentModal !== ModalGroupOption.ViewScriptDetails}
        showHostScriptActions
        teamIdForApi={teamIdForApi}
      />
      <DeleteScriptModal
        scriptId={selectedScript?.script_id || 1}
        scriptName={selectedScript?.name || ""}
        onCancel={() => {
          setCurrentModal(previousModal);
          setPreviousModal(ModalGroupOption.Run);
        }}
        afterDelete={() => {
          // The delete API call is handled in DeleteScriptModal
          setCurrentModal(ModalGroupOption.Run);
          setPreviousModal(null);
          refetchHostScripts();
          setSelectedScript(null);
        }}
        isHidden={currentModal !== ModalGroupOption.Delete}
      />
      <RunScriptDetailsModal
        scriptExecutionId={selectedExecutionId || ""}
        onCancel={() => {
          if (previousModal === ModalGroupOption.ViewScriptDetails) {
            setCurrentModal(previousModal);
            setPreviousModal(ModalGroupOption.Run);
          } else if (previousModal === ModalGroupOption.Run) {
            setCurrentModal(previousModal);
            setPreviousModal(null);
          }
        }}
        isHidden={currentModal !== ModalGroupOption.ViewRunDetails}
      />
    </>
  );
};

export default ScriptModalGroup;
