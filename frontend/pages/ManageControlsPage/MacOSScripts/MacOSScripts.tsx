import React, { useRef, useState } from "react";

import { IMdmScript } from "interfaces/mdm";

import CustomLink from "components/CustomLink";

import ScriptListHeading from "./components/ScriptListHeading";
import ScriptListItem from "./components/ScriptListItem";
import DeleteScriptModal from "./components/DeleteScriptModal";
import FileUploader from "../components/FileUploader";
import UploadList from "../components/UploadList";
import RerunScriptModal from "./components/RerunScriptModal";

// TODO: remove when get integrate with API.
const scripts = [
  {
    id: 1,
    name: "Test.py",
    ran: 57,
    pending: 2304,
    errors: 0,
    created_at: new Date().toString(),
  },
];

const baseClass = "mac-os-scripts";

const MacOSScripts = () => {
  const [showRerunScriptModal, setShowRerunScriptModal] = useState(false);
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const selectedScript = useRef<IMdmScript | null>(null);

  const onClickRerun = (script: IMdmScript) => {
    selectedScript.current = script;
    setShowRerunScriptModal(true);
  };

  const onClickDelete = (script: IMdmScript) => {
    selectedScript.current = script;
    setShowDeleteScriptModal(true);
  };

  const onCancelRerun = () => {
    selectedScript.current = null;
    setShowRerunScriptModal(false);
  };

  const onCancelDelete = () => {
    selectedScript.current = null;
    setShowDeleteScriptModal(false);
  };

  // TODO: change when integrating with API
  const onRerunScript = (scriptId: number) => {
    console.log("rerun", scriptId);
    setShowRerunScriptModal(false);
  };

  // TODO: change when integrating with API
  const onDeleteScript = (scriptId: number) => {
    console.log("delete", scriptId);
    setShowDeleteScriptModal(false);
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to change configuration and remediate issues on macOS
        hosts. Each script runs once per host. All scripts can be rerun on end
        users’ My device page. <CustomLink text="Learn more" url="#" newTab />
      </p>
      <UploadList
        listItems={scripts}
        HeadingComponent={ScriptListHeading}
        ListItemComponent={({ listItem }) => (
          <ScriptListItem
            script={listItem}
            onRerun={onClickRerun}
            onDelete={onClickDelete}
          />
        )}
      />
      <FileUploader
        icon="files"
        message="Any type of script supported by macOS. If you If you don’t specify a shell or interpreter (e.g. #!/bin/sh), the script will run in /bin/sh."
        onFileUpload={() => {
          return null;
        }}
      />
      {showRerunScriptModal && selectedScript.current && (
        <RerunScriptModal
          scriptName={selectedScript.current?.name}
          scriptId={selectedScript.current?.id}
          onCancel={onCancelRerun}
          onRerun={onRerunScript}
        />
      )}
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

export default MacOSScripts;
