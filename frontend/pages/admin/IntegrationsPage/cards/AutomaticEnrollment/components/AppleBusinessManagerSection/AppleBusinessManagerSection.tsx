import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import FileSaver from "file-saver";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { IMdmAppleBm } from "interfaces/mdm";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { readableDate } from "utilities/helpers";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import TooltipWrapper from "components/TooltipWrapper";
import DataError from "components/DataError";
import Spinner from "components/Spinner/Spinner";
import SectionHeader from "components/SectionHeader";

import EditTeamModal from "../EditTeamModal";
import WindowsAutomaticEnrollmentCard from "./components/WindowsAutomaticEnrollmentCard/WindowsAutomaticEnrollmentCard";

const baseClass = "apple-business-manager-section";

interface IABMKeys {
  decodedPublic: string;
  decodedPrivate: string;
}

interface IAppleBusinessManagerSectionProps {
  router: InjectedRouter;
}

const AppleBusinessManagerSection = ({
  router,
}: IAppleBusinessManagerSectionProps) => {
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);
  const [defaultTeamName, setDefaultTeamName] = useState("No team");
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const {
    data: mdmAppleBm,
    isLoading: isLoadingMdmAppleBm,
    error: errorMdmAppleBm,
  } = useQuery<IMdmAppleBm, AxiosError, IMdmAppleBm>(
    ["mdmAppleBmAPI"],
    () => mdmAppleBmAPI.getAppleBMInfo(),
    {
      refetchOnWindowFocus: false,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
      onSuccess: (appleBmData) => {
        setDefaultTeamName(appleBmData.default_team ?? "No team");
      },
    }
  );

  const {
    data: keys,
    error: fetchKeysError,
    isFetching: isFetchingKeys,
  } = useQuery<IABMKeys, Error>(["keys"], () => mdmAppleBmAPI.loadKeys(), {
    refetchOnWindowFocus: false,
    retry: false,
  });

  const toggleEditTeamModal = () => {
    setShowEditTeamModal(!showEditTeamModal);
  };

  const navigateToWindowsAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  const onDownloadKeys = (evt: React.MouseEvent) => {
    evt.preventDefault();

    // MDM TODO: Confirm error flash message
    if (isFetchingKeys || fetchKeysError) {
      renderFlash(
        "error",
        "Your MDM business manager keys could not be downloaded. Please try again."
      );
      return false;
    }

    if (keys) {
      const publicFilename = "fleet-apple-mdm-bm-public-key.crt";
      const publicFile = new global.window.File(
        [keys.decodedPublic],
        publicFilename,
        {
          type: "application/x-pem-file",
        }
      );

      const privateFilename = "fleet-apple-mdm-bm-private.key";
      const privateFile = new global.window.File(
        [keys.decodedPrivate],
        privateFilename,
        {
          type: "application/x-pem-file",
        }
      );

      FileSaver.saveAs(publicFile);
      setTimeout(() => {
        FileSaver.saveAs(privateFile);
      }, 100);
    } else {
      renderFlash(
        "error",
        "Your MDM business manager keys could not be downloaded. Please try again."
      );
    }
    return false;
  };

  const renderAppleBMInfo = () => {
    // we want to give a more useful error message for 400s.
    if (errorMdmAppleBm && errorMdmAppleBm.status === 400) {
      return (
        <DataError>
          <span className={`${baseClass}__400-error-info`}>
            The Apple Business Manager certificate or server token is invalid.
            Restart Fleet with a valid certificate and token.
          </span>
          <span className={`${baseClass}__400-error-info`}>
            See our{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/setup-abm"
              text="ABM documentation"
              newTab
            />{" "}
            for help.
          </span>
        </DataError>
      );
    }

    // The API returns a 404 error if ABM is not configured yet, in that case we
    // want to prompt the user to download the certs and keys to configure the
    // server instead of the default error message.
    const showMdmAppleBmError =
      errorMdmAppleBm && errorMdmAppleBm.status !== 404;

    if (showMdmAppleBmError) {
      return <DataError />;
    }

    // no error, but no apple bm data yet. TODO: when does this happen?
    if (!mdmAppleBm) {
      return (
        <>
          <div className={`${baseClass}__section-description`}>
            Connect Fleet to your Apple Business Manager account to
            automatically enroll macOS hosts to Fleet when they&apos;re first
            setup.
          </div>
          <div className={`${baseClass}__section-instructions`}>
            <p>1. Download your public and private keys.</p>
            <Button onClick={onDownloadKeys} variant="brand">
              Download
            </Button>
            <p>
              2. Sign in to{" "}
              <CustomLink
                url="https://business.apple.com/"
                text="Apple Business Manager"
                newTab
              />
              <br />
              If your organization doesn&apos;t have an account, select{" "}
              <b>Enroll now</b>.
            </p>
            <p>
              3. In Apple Business Manager, upload your public key and download
              your server token.
            </p>
            <p>
              4. Deploy Fleet with <b>mdm</b> configuration.{" "}
              <CustomLink
                url="https://fleetdm.com/docs/deploying/configuration#mobile-device-management-mdm"
                text="See how"
                newTab
              />
            </p>
          </div>
        </>
      );
    }

    // we have the apple bm data and render it
    return (
      <>
        <div className={`${baseClass}__section-description`}>
          To use automatically enroll macOS hosts to Fleet when they’re first
          unboxed, Apple Inc. requires a server token.
        </div>
        <div className={`${baseClass}__section-information`}>
          <h4>
            <TooltipWrapper tipContent="macOS hosts will be added to this team when they’re first unboxed.">
              Team
            </TooltipWrapper>
          </h4>
          <p>
            {defaultTeamName}{" "}
            <Button
              className={`${baseClass}__edit-team-btn`}
              onClick={toggleEditTeamModal}
              variant="text-icon"
            >
              Edit <Icon name="pencil" />
            </Button>
          </p>
          <h4>Apple ID</h4>
          <p>{mdmAppleBm.apple_id}</p>
          <h4>Organization name</h4>
          <p>{mdmAppleBm.org_name}</p>
          <h4>MDM server URL</h4>
          <p>{mdmAppleBm.mdm_server_url}</p>
          <h4>Renew date</h4>
          <p>{readableDate(mdmAppleBm.renew_date)}</p>
        </div>
      </>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Apple Business Manager" />
      {isLoadingMdmAppleBm ? <Spinner /> : renderAppleBMInfo()}
      <WindowsAutomaticEnrollmentCard
        viewDetails={navigateToWindowsAutomaticEnrollment}
      />
      {showEditTeamModal && (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          defaultTeamName={defaultTeamName}
          onUpdateSuccess={(newDefaultTeamName) =>
            setDefaultTeamName(newDefaultTeamName)
          }
        />
      )}
    </div>
  );
};

export default AppleBusinessManagerSection;
