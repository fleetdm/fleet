import React from "react";

import CustomLink from "components/CustomLink";

import FileUploader from "../components/FileUploader";
import UploadList from "../components/UploadList";

const baseClass = "mac-os-scripts";

interface IMacOSScriptsProps {}

const MacOSScripts = ({}: IMacOSScriptsProps) => {
  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Upload scripts to change configuration and remediate issues on macOS
        hosts. Each script runs once per host. All scripts can be rerun on end
        users’ My device page. <CustomLink text="Learn more" url="#" newTab />
      </p>
      {/* <UploadList
        listItems={[1, 2, 3]}
        HeadingComponent={() => <span>Header</span>}
        ItemComponent={() => <p>item</p>}
      /> */}
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
