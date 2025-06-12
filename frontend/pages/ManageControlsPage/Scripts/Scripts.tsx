import { AxiosError } from "axios";
import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { IScript } from "interfaces/script";
import PATHS from "router/paths";
import scriptAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";

import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import InfoBanner from "components/InfoBanner";
import Spinner from "components/Spinner";
import Pagination from "components/Pagination";
import UploadList from "../components/UploadList";
import DeleteScriptModal from "./components/DeleteScriptModal";
import EditScriptModal from "./components/EditScriptModal";
import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import ScriptUploader from "./components/ScriptUploader";

const baseClass = "scripts";

const SCRIPTS_PER_PAGE = 10;

interface IScriptsProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Scripts = ({ router, currentPage, teamIdForApi }: IScriptsProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);
  const [showEditScripsModal, setShowEditScriptModal] = useState(false);

  const selectedScript = useRef<IScript | null>(null);

  const {
    data: { scripts, meta } = {},
    isLoading,
    isError,
    refetch: refetchScripts,
  } = useQuery<
    IScriptsResponse,
    AxiosError,
    IScriptsResponse,
    IListScriptsQueryKey[]
  >(
    [
      {
        scope: "scripts",
        team_id: teamIdForApi,
        page: currentPage,
        per_page: SCRIPTS_PER_PAGE,
      },
    ],
    ({ queryKey: [{ team_id, page, per_page }] }) =>
      scriptAPI.getScripts({ team_id, page, per_page }),
    {
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 3000,
    }
  );

  // pagination controls
  const path = PATHS.CONTROLS_SCRIPTS;
  const queryString = isPremiumTier ? `?team_id=${teamIdForApi}&` : "?";
  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);
  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  const { config } = useContext(AppContext);
  if (!config) return null;

  const onClickScript = (script: IScript) => {
    selectedScript.current = script;
    setShowEditScriptModal(true);
  };

  const onEditScript = (script: IScript) => {
    selectedScript.current = script;
    setShowEditScriptModal(true);
  };

  const onExitEditScript = () => {
    selectedScript.current = null;
    setShowEditScriptModal(false);
  };

  const onClickDelete = (script: IScript) => {
    selectedScript.current = script;
    setShowDeleteScriptModal(true);
  };

  const onCancelDelete = () => {
    setShowDeleteScriptModal(false);
    selectedScript.current = null;
  };

  const onDeleteScript = () => {
    selectedScript.current = null;
    setShowDeleteScriptModal(false);
    refetchScripts();
  };

  const onUploadScript = () => {
    refetchScripts();
  };

  const renderScriptsList = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (currentPage === 0 && !scripts?.length) {
      return null;
    }

    return (
      <>
        <UploadList
          keyAttribute="id"
          listItems={scripts || []}
          HeadingComponent={ScriptListHeading}
          ListItemComponent={({ listItem }) => (
            <ScriptListItem
              script={listItem}
              onDelete={onClickDelete}
              onClickScript={onClickScript}
              onEdit={onEditScript}
            />
          )}
        />
        <Pagination
          disablePrev={isLoading || !meta?.has_previous_results}
          disableNext={isLoading || !meta?.has_next_results}
          hidePagination={
            !isLoading && !meta?.has_previous_results && !meta?.has_next_results
          }
          onPrevPage={onPrevPage}
          onNextPage={onNextPage}
        />
      </>
    );
  };

  const renderScriptsDisabledBanner = () => (
    <InfoBanner color="yellow">
      <div>
        <b>Running scripts is disabled in organization settings.</b> You can
        still manage your library of macOS and Windows scripts below.
      </div>
    </InfoBanner>
  );

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to remediate issues on macOS, Windows, and Linux hosts.
        You can run scripts on individual hosts.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/docs/using-fleet/scripts"
          newTab
        />
      </p>

      {config.server_settings.scripts_disabled && renderScriptsDisabledBanner()}
      {renderScriptsList()}
      <ScriptUploader currentTeamId={teamIdForApi} onUpload={onUploadScript} />
      {showDeleteScriptModal && selectedScript.current && (
        <DeleteScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelDelete}
          onDone={onDeleteScript}
        />
      )}
      {showEditScripsModal && selectedScript.current && (
        <EditScriptModal
          scriptId={selectedScript.current.id}
          scriptName={selectedScript.current.name}
          onExit={onExitEditScript}
        />
      )}
    </div>
  );
};

export default Scripts;
