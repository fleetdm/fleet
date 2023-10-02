import React, { useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import scriptAPI, {
  IScript,
  IScriptsResponse,
} from "services/entities/scripts";

import CustomLink from "components/CustomLink";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import DeleteScriptModal from "./components/DeleteScriptModal";
import FileUploader from "../components/FileUploader";
import UploadList from "../components/UploadList";
import ScriptUploader from "./components/ScriptUploader";

const baseClass = "scripts";

interface IScriptsProps {
  teamIdForApi: number;
}

const Scripts = ({ teamIdForApi }: IScriptsProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const selectedScript = useRef<IScript | null>(null);

  const {
    data: scripts,
    isLoading,
    isError,
    refetch: refetchScripts,
  } = useQuery<IScriptsResponse, AxiosError, IScript[]>(
    ["scripts"],
    () => scriptAPI.getScripts(),
    {
      retry: false,
      refetchOnWindowFocus: false,
      select: (data) => data.scripts,
    }
  );

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
  };

  const onUploadScript = () => {
    refetchScripts();
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
      {scripts && scripts.length !== 0 && (
        <UploadList
          listItems={scripts}
          HeadingComponent={ScriptListHeading}
          ListItemComponent={({ listItem }) => (
            <ScriptListItem script={listItem} onDelete={onClickDelete} />
          )}
        />
      )}
      <ScriptUploader currentTeamId={teamIdForApi} onUpload={onUploadScript} />
      {showDeleteScriptModal && selectedScript.current && (
        <DeleteScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelDelete}
          onDelete={onDeleteScript}
        />
      )}
    </div>
  );
};

export default Scripts;
