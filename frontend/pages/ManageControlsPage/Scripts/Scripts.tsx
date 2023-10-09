import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import scriptAPI, {
  IListScriptsQueryKey,
  IScript,
  IScriptsResponse,
} from "services/entities/scripts";

import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import ScriptListPagination from "./components/ScriptListPagination";
import DeleteScriptModal from "./components/DeleteScriptModal";
import UploadList from "../components/UploadList";
import ScriptUploader from "./components/ScriptUploader";

const baseClass = "scripts";

const SCRIPTS_PER_PAGE = 2; // TODO: confirm this is the desired default

interface IScriptsProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Scripts = ({ router, currentPage, teamIdForApi }: IScriptsProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

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

  const path = PATHS.CONTROLS_SCRIPTS.concat(`?team_id=${teamIdForApi}`);
  const onPrevPage = useCallback(() => {
    router.push(path.concat(`&page=${currentPage - 1}`));
  }, [router, path, currentPage]);
  const onNextPage = useCallback(() => {
    router.push(path.concat(`&page=${currentPage + 1}`));
  }, [router, path, currentPage]);

  // The user is not a premium tier, so show the premium feature message.
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  const onClickDelete = (script: IScript) => {
    selectedScript.current = script;
    setShowDeleteScriptModal(true);
  };

  const onCancelDelete = () => {
    selectedScript.current = null;
    setShowDeleteScriptModal(false);
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
          listItems={scripts || []}
          HeadingComponent={ScriptListHeading}
          ListItemComponent={({ listItem }) => (
            <ScriptListItem script={listItem} onDelete={onClickDelete} />
          )}
        />
        <ScriptListPagination
          meta={meta}
          isLoading={isLoading}
          onPrevPage={onPrevPage}
          onNextPage={onNextPage}
        />
      </>
    );
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to change configuration and remediate issues on macOS
        hosts. You can run scripts on individual hosts.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/docs/using-fleet/scripts"
          newTab
        />
      </p>
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
    </div>
  );
};

export default Scripts;
