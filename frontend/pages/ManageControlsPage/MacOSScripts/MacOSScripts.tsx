import React from "react";
import FileUploader from "../components/FileUploader";

const baseClass = "mac-os-scripts";

interface IMacOSScriptsProps {}

const MacOSScripts = ({}: IMacOSScriptsProps) => {
  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of macOS updates on hosts assigned
        to this team.
      </p>
      <FileUploader
        icon="files"
        message="Any type of script supported by macOS. If you If you don’t specify a shell or interpreter (e.g. #!/bin/sh), the script will run in /bin/sh."
        onFileUpload={() => {
          return null;
        }}
      />
    </div>
  );
};

export default MacOSScripts;
