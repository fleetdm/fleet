import React, { useContext } from "react";
import FileSaver from "file-saver";

import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";
import { IDeviceUserResponse } from "interfaces/host";
import { AxiosResponse } from "axios";

interface IManualEnrollMdmModalProps {
  host: IDeviceUserResponse["host"];
  onCancel: () => void;
  token?: string;
}

const baseClass = "manual-enroll-mdm-modal";

const ManualEnrollMdmModal = ({
  host: { platform, os_version },
  onCancel,
  token = "",
}: IManualEnrollMdmModalProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const onDownload = async () => {
    try {
      const profileContent = await mdmAPI.downloadManualEnrollmentProfile(
        token
      );
      console.log(profileContent);
      const file = new File(
        [profileContent],
        "fleet-mdm-enrollment-profile.mobileconfig"
      );
      FileSaver.saveAs(file);
    } catch (e: AxiosResponse | unknown) {
      // We need to do some additional parsing here, as the data is a blob.
      if (e && typeof e === "object" && "data" in e) {
        const axiosResponse = e as AxiosResponse;
        if (axiosResponse.data) {
          console.log(axiosResponse);
          const dataBlob = axiosResponse.data as Blob;
          const blobText = await dataBlob.text();
          console.log("blob text", blobText);
          if (
            blobText.includes(
              "The team associated with the enroll_secret has end user authentication enabled so the OTA profile won't work."
            )
          ) {
            renderFlash(
              "error",
              "The team associated with the enroll_secret has end user authentication enabled so the enrollment profile doesn't work. Please contact your IT admin."
            );
            return;
          }
        }
      }

      renderFlash("error", "Failed to download the profile. Please try again.");
    }
  };

  let isMacOsSequoiaOrLater = false;
  if (platform === "darwin" && os_version.startsWith("macOS ")) {
    const [major] = os_version
      .replace("macOS ", "")
      .split(".")
      .map((s) => parseInt(s, 10));
    isMacOsSequoiaOrLater = major >= 15;
  }

  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you download and install a
          profile.
        </p>
        <ol>
          <li>
            <span>Download your profile.</span>
            <br />
            {/* TODO: make a link component that appears as a button. */}
            <Button
              className={`${baseClass}__download-button`}
              onClick={onDownload}
            >
              Download
            </Button>
          </li>
          <li>Open the profile you just downloaded.</li>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b>.
          </li>
          <li>
            {isMacOsSequoiaOrLater ? (
              <>
                In the sidebar menu, select <b>Profile Downloaded</b>, find and
                double-click the <b>[Organization name] enrollment</b> profile.
              </>
            ) : (
              <>
                In the search bar, type “Profiles”. Select <b>Profiles</b>, find
                and double click the <b>[Organization name] enrollment</b>{" "}
                profile.
              </>
            )}
          </li>
          <li>
            Select <b>Install</b> then enter your password.
          </li>
          <li>
            Select <b>Done</b> to close this window and select <b>Refetch</b> on
            your My device page to tell your organization that MDM is on.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel}>
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManualEnrollMdmModal;
