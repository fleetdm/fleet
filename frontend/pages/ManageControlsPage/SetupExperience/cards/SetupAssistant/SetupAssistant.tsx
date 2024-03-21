import React, { useState } from "react";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";

import SetupAssistantPreview from "./components/SetupAssistantPreview";
import SetupAssistantPackageUploader from "./components/SetupAssistantPackageUploader";
import SetuAssistantUploadedProfileView from "./components/SetupAssistantUploadedProfileView/SetupAssistantUploadedProfileView";
import DeleteAutoEnrollmentProfile from "./components/DeleteAutoEnrollmentProfile";

const baseClass = "setup-assistant";

interface ISetupAssistantProps {
  currentTeamId: number;
}

const StartupAssistant = ({ currentTeamId }: ISetupAssistantProps) => {
  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);

  const isLoading = false;

  const noPackageUploaded = true;

  const onDelete = () => {};

  return (
    <div className={baseClass}>
      <SectionHeader title="Setup assistant" />
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          {false ? (
            <>
              <SetupAssistantPackageUploader
                currentTeamId={currentTeamId}
                onUpload={() => 1}
              />
              <div className={`${baseClass}__preview-container`}>
                <SetupAssistantPreview />
              </div>
            </>
          ) : (
            <>
              <SetuAssistantUploadedProfileView
                profileMetaData={1}
                currentTeamId={currentTeamId}
                onDelete={() => setShowDeleteProfileModal(true)}
              />
              <div className={`${baseClass}__preview-container`}>
                <SetupAssistantPreview />
              </div>
            </>
          )}
        </div>
      )}
      {showDeleteProfileModal && (
        <DeleteAutoEnrollmentProfile
          onDelete={onDelete}
          onCancel={() => setShowDeleteProfileModal(false)}
        />
      )}
    </div>
  );
};

export default StartupAssistant;
