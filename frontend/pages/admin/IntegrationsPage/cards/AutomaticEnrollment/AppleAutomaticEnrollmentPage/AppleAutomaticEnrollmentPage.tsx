import React, { useCallback, useContext, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";
import { IMdmAppleBm } from "interfaces/mdm";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { readableDate } from "utilities/helpers";

import BackLink from "components/BackLink";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink/CustomLink";
import DataError from "components/DataError";
import FileUploader from "components/FileUploader";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";

// TODO: Implement DownloadKey functioonality, for now it is using the DownloadCSR component as a placeholder
import DownloadKey from "../../MdmSettings/MacOSMdmPage/components/actions/DownloadCSR";
import DisableAutomaticEnrollmentModal from "./modals/DisableAutomaticEnrollmentModal";
import RenewTokenModal from "./modals/RenewTokenModal";

const baseClass = "apple-automatic-enrollment-page";

const AppleAutomaticEnrollmentPage = ({
  router,
}: {
  router: InjectedRouter;
}) => {
  // TODO: implement uploading state
  const [isUploading, setIsUploading] = useState(false);
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showRenewModal, setShowRenewModal] = useState(false);

  const {
    data: mdmAppleBm,
    isLoading: isLoadingMdmAppleBm,
    error: errorMdmAppleBm,
    refetch,
  } = useQuery<IMdmAppleBm, AxiosError, IMdmAppleBm>(
    ["mdmAppleBmAPI"],
    () => mdmAppleBmAPI.getAppleBMInfo(),
    {
      refetchOnWindowFocus: false,
    }
  );

  const onConfirmDisable = useCallback(() => {
    // TODO: Implement this
    console.log("Disable automatic enrollment");
    refetch();
    setShowDisableModal(false);
  }, [refetch]);

  if (isLoadingMdmAppleBm) {
    return (
      <div className={baseClass}>
        <Spinner />
      </div>
    );
  }

  if (errorMdmAppleBm?.status === 404) {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  }

  if (errorMdmAppleBm) {
    return <DataError />;
  }

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to automatic enrollment"
          path={PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT}
          className={`${baseClass}__back-to-automatic-enrollment`}
        />
        <h1>Apple Business Manager (ABM)</h1>
        {mdmAppleBm ? (
          <div>
            <h4>Apple ID</h4>
            <p>{mdmAppleBm.apple_id}</p>
            <h4>Organization name</h4>
            <p>{mdmAppleBm.org_name}</p>
            <h4>MDM server URL</h4>
            <p>{mdmAppleBm.mdm_server_url}</p>
            <h4>Renew date</h4>
            <p>{readableDate(mdmAppleBm.renew_date)}</p>
            <div className={`${baseClass}__button-wrap`}>
              <Button
                variant="inverse"
                onClick={() => setShowDisableModal(true)}
              >
                Disable automatic enrollment
              </Button>
              <Button variant="brand" onClick={() => setShowRenewModal(true)}>
                Renew token
              </Button>
            </div>
          </div>
        ) : (
          <>
            <p>
              Connect Fleet to your Apple Business Manager account to
              automatically enroll macOS hosts to Fleet when they’re first
              booted.{" "}
            </p>
            {/* Ideally we'd use the native browser list styles and css to display
        the list numbers but this does not allow us to style the list items as we'd
        like so we write the numbers in the JSX instead. */}
            <ol className={`${baseClass}__setup-list`}>
              <li>
                <span>1.</span>
                <p>
                  Download your public key.{" "}
                  <DownloadKey baseClass={baseClass} />
                </p>
              </li>
              <li>
                <span>2.</span>
                <span>
                  <span>
                    Sign in to{" "}
                    <CustomLink
                      newTab
                      text="Apple Business Manager"
                      url="https://business.apple.com"
                    />
                    <br />
                    If your organization doesn’t have an account, select{" "}
                    <b>Enroll now</b>.
                  </span>
                </span>
              </li>
              <li>
                <span>3.</span>
                <span>
                  Select your <b>account name</b> at the bottom left of the
                  screen, then select <b>Preferences</b>.
                </span>
              </li>
              <li>
                <span>4.</span>
                <span>
                  In the <b>Your MDM Servers</b> section, select <b>Add</b>.
                </span>
              </li>
              <li>
                <span>5.</span>
                <span>Enter a name for the server such as “Fleet”.</span>
              </li>
              <li>
                <span>6.</span>
                <span>
                  Under <b>MDM Server Settings</b>, upload the public key
                  downloaded in the first step and select <b>Save</b>.
                </span>
              </li>
              <li>
                <span>7.</span>
                <span>
                  In the <b>Default Device Assignment</b> section, select{" "}
                  <b>Change</b>, then assign the newly created server as the
                  default for your Macs, and select <b>Done</b>.
                </span>
              </li>
              <li>
                <span>8.</span>
                <span>
                  Select newly created server in the sidebar, then select{" "}
                  <b>Download Token</b> on the top.
                </span>
              </li>
              <li>
                <span>9.</span>
                <span>Upload the downloaded token (.p7m file).</span>
              </li>
            </ol>
            <FileUploader
              className={`${baseClass}__file-uploader ${
                isUploading ? `${baseClass}__file-uploader--loading` : ""
              }`}
              accept=".p7m"
              message="ABM token (.p7m)"
              graphicName={"file-p7m"}
              buttonType="link"
              buttonMessage={isUploading ? "Uploading..." : "Upload"}
              onFileUpload={() => console.log("file uploaded")}
            />
          </>
        )}
      </>
      {showDisableModal && (
        <DisableAutomaticEnrollmentModal
          onCancel={() => setShowDisableModal(false)}
          onConfirm={onConfirmDisable}
        />
      )}
      {showRenewModal && (
        <RenewTokenModal onCancel={() => setShowRenewModal(false)} />
      )}
    </MainContent>
  );
};

export default AppleAutomaticEnrollmentPage;
