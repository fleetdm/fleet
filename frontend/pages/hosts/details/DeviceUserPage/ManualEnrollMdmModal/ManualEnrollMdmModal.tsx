import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import FileSaver from "file-saver";

import { NotificationContext } from "context/notification";

import DataError from "components/DataError";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Spinner from "components/Spinner";

import mdmAPI from "services/entities/mdm";

interface IManualEnrollMdmModalProps {
  onCancel: () => void;
  token?: string;
}

const baseClass = "manual-enroll-mdm-modal enroll-mdm-modal";

const ManualEnrollMdmModal = ({
  onCancel,
  token = "",
}: IManualEnrollMdmModalProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const [isDownloadingProfile, setIsDownloadingProfile] = useState(false);

  const {
    data: enrollmentProfile,
    error: fetchMdmProfileError,
    isFetching: isFetchingMdmProfile,
  } = useQuery<string, Error>(
    ["enrollment profile"],
    () => mdmAPI.downloadDeviceUserEnrollmentProfile(token),
    {
      refetchOnWindowFocus: false,
    }
  );

  const onDownloadProfile = (evt: React.MouseEvent) => {
    evt.preventDefault();
    setIsDownloadingProfile(true);

    setTimeout(() => setIsDownloadingProfile(false), 1000);

    if (enrollmentProfile) {
      const filename = "fleet-mdm-enrollment-profile.mobileconfig";
      const file = new global.window.File([enrollmentProfile], filename, {
        type: "application/x-apple-aspen-config",
      });

      FileSaver.saveAs(file);
    } else {
      renderFlash(
        "error",
        "Your enrollment profile could not be downloaded. Please try again."
      );
    }

    return false;
  };
  if (isFetchingMdmProfile) {
    return <Spinner />;
  }
  if (fetchMdmProfileError) {
    return <DataError card />;
  }

  return (
    <Modal title="Turn on MDM" onExit={onCancel} className={baseClass}>
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you download and install a
          profile.
        </p>
        <ol>
          <li>
            {!isFetchingMdmProfile && (
              <>
                <span>Download your profile.</span>
              </>
            )}
            {fetchMdmProfileError ? (
              <span className={`${baseClass}__error`}>
                {fetchMdmProfileError}
              </span>
            ) : (
              <Button
                type="button"
                onClick={onDownloadProfile}
                variant="brand"
                isLoading={isDownloadingProfile}
                className={`${baseClass}__download-button`}
              >
                Download
              </Button>
            )}
          </li>
          <li>Open the profile you just downloaded.</li>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b> or <b>System Preferences</b>.
          </li>
          <li>
            In the search bar, type “Profiles”. Select <b>Profiles</b>, double
            click <b>Enrollment Profile</b>, and select <b>Install</b>.
          </li>
          <li>
            Select <b>Enroll</b> then enter your password.
          </li>
          <li>
            Close this window and select <b>Refetch</b> on your My device page
            to tell your organization that MDM is on.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManualEnrollMdmModal;
