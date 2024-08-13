import React, { useCallback, useContext, useState } from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
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

import DownloadKey from "../../../../components/DownloadFileButtons/DownloadABMKey";
import DisableAutomaticEnrollmentModal from "./modals/DisableAutomaticEnrollmentModal";
import RenewTokenModal from "./modals/RenewTokenModal";

const baseClass = "apple-business-manager-page";

const ButtonWrap = ({
  onClickDisable,
  onClickRenew,
}: {
  onClickDisable: () => void;
  onClickRenew: () => void;
}) => {
  return (
    <div className={`${baseClass}__button-wrap`}>
      <Button variant="inverse" onClick={onClickDisable}>
        Disable automatic enrollment
      </Button>
      <Button variant="brand" onClick={onClickRenew}>
        Renew token
      </Button>
    </div>
  );
};

const AppleBusinessManagerPage = ({ router }: { router: InjectedRouter }) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUploading, setIsUploading] = useState(false);
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showRenewModal, setShowRenewModal] = useState(false);

  const {
    data: mdmAppleBm,
    error: errorMdmAppleBm,
    isLoading,
    isRefetching,
    refetch,
  } = useQuery<IMdmAppleBm, AxiosError, IMdmAppleBm>(
    ["mdmAppleBmAPI"],
    () => mdmAppleBmAPI.getAppleBMInfo(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
    }
  );

  const uploadToken = useCallback(
    async (data: FileList | null) => {
      setIsUploading(true);
      const token = data?.[0];
      if (!token) {
        setIsUploading(false);
        renderFlash("error", "No token selected.");
        return;
      }

      try {
        await mdmAppleBmAPI.uploadToken(token);
        renderFlash(
          "success",
          "Automatic enrollment for macOS hosts is enabled."
        );
        router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT);
      } catch (e) {
        const msg = getErrorReason(e);
        if (msg.toLowerCase().includes("valid token")) {
          renderFlash("error", msg);
        } else {
          renderFlash("error", "Couldn’t enable. Please try again.");
        }
      } finally {
        setIsUploading(false);
      }
    },
    [renderFlash, router]
  );

  const onClickDisable = useCallback(() => {
    setShowDisableModal(true);
  }, []);

  const onClickRenew = useCallback(() => {
    setShowRenewModal(true);
  }, []);

  const disableAutomaticEnrollment = useCallback(async () => {
    try {
      await mdmAppleBmAPI.disableAutomaticEnrollment();
      renderFlash("success", "Automatic enrollment disabled successfully.");
      router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT);
    } catch (e) {
      renderFlash(
        "error",
        "Couldn’t disable automatic enrollment. Please try again."
      );
      setShowDisableModal(false);
    }
  }, [renderFlash, router]);

  const onCancelDisable = useCallback(() => {
    setShowDisableModal(false);
  }, []);

  const onRenew = useCallback(() => {
    refetch();
    setShowRenewModal(false);
  }, [refetch]);

  const onCancelRenew = useCallback(() => {
    setShowRenewModal(false);
  }, []);

  if (isLoading || isRefetching) {
    return <Spinner />;
  }

  const showDataError = errorMdmAppleBm && errorMdmAppleBm.status !== 404;
  const showConnectAbm = !mdmAppleBm;

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to automatic enrollment"
          path={PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT}
          className={`${baseClass}__back-to-automatic-enrollment`}
        />
        <h1>Apple Business Manager (ABM)</h1>
        {showDataError && (
          <div>
            <DataError />
            <ButtonWrap
              onClickDisable={onClickDisable}
              onClickRenew={onClickRenew}
            />
          </div>
        )}
        {!showDataError && !showConnectAbm && (
          <div>
            <h4>Apple ID</h4>
            <p>{mdmAppleBm.apple_id}</p>
            <h4>Organization name</h4>
            <p>{mdmAppleBm.org_name}</p>
            <h4>MDM server URL</h4>
            <p>{mdmAppleBm.mdm_server_url}</p>
            <h4>Renew date</h4>
            <p>{readableDate(mdmAppleBm.renew_date)}</p>
            <ButtonWrap
              onClickDisable={onClickDisable}
              onClickRenew={onClickRenew}
            />
          </div>
        )}
        {!showDataError && showConnectAbm && (
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
                    <b>Sign up now</b>.
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
              onFileUpload={uploadToken}
            />
          </>
        )}
      </>
      {showDisableModal && (
        <DisableAutomaticEnrollmentModal
          onCancel={onCancelDisable}
          onConfirm={disableAutomaticEnrollment}
        />
      )}
      {showRenewModal && (
        <RenewTokenModal onCancel={onCancelRenew} onRenew={onRenew} />
      )}
    </MainContent>
  );
};

export default AppleBusinessManagerPage;
