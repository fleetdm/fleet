import React, { useCallback, useContext, useRef, useState } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import PATHS from "router/paths";

import { AppContext } from "context/app";

import { IScript } from "interfaces/script";
import scriptAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";

import DataError from "components/DataError";
import InfoBanner from "components/InfoBanner";
import Spinner from "components/Spinner";
import Pagination from "components/Pagination";
import SectionHeader from "components/SectionHeader";

import UploadList from "../../../components/UploadList";
import DeleteScriptModal from "../../components/DeleteScriptModal";
import EditScriptModal from "../../components/EditScriptModal";
import ScriptListHeading from "../../components/ScriptListHeading";
import ScriptListItem from "../../components/ScriptListItem";
import ScriptUploader from "../../components/ScriptUploader";
import { IScriptsCommonProps } from "../../ScriptsNavItems";

const baseClass = "script-library";

const SCRIPTS_PER_PAGE = 10;

export interface IScriptLibraryProps extends IScriptsCommonProps {
  currentPage?: number;
}

const ScriptLibrary = ({
  router,
  teamId,
  currentPage = 0,
}: IScriptLibraryProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);
  const [showEditScriptModal, setShowEditScriptModal] = useState(false);

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
        team_id: teamId,
        page: currentPage,
        per_page: SCRIPTS_PER_PAGE,
      },
    ],
    ({ queryKey: [{ team_id, page, per_page }] }) =>
      scriptAPI.getScripts({ team_id, page, per_page }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 3000,
    }
  );

  // pagination controls
  const path = PATHS.CONTROLS_SCRIPTS_LIBRARY;
  const queryString = isPremiumTier ? `?team_id=${teamId}&` : "?";
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
      <SectionHeader title="Library" alignLeftHeaderVertically />
      {config.server_settings.scripts_disabled && renderScriptsDisabledBanner()}
      {renderScriptsList()}
      <ScriptUploader currentTeamId={teamId} onUpload={onUploadScript} />
      {showDeleteScriptModal && selectedScript.current && (
        <DeleteScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelDelete}
          onDone={onDeleteScript}
        />
      )}
      {showEditScriptModal && selectedScript.current && (
        <EditScriptModal
          scriptId={selectedScript.current.id}
          scriptName={selectedScript.current.name}
          onExit={onExitEditScript}
        />
      )}
    </div>
  );
};

export default ScriptLibrary;
