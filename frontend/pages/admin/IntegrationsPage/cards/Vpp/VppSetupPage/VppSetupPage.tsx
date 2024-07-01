import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import MainContent from "components/MainContent";
import BackLink from "components/BackLink";
import FileUploader from "components/FileUploader";
import DataSet from "components/DataSet";
import Button from "components/buttons/Button";

import DisableVppModal from "./components/DisableVppModal";
import VppSetupSteps from "./components/VppSetupSteps";
import RenewVppTokenModal from "./components/RenewVppTokenModal";

const baseClass = "vpp-setup-page";

interface IVppSetupContentProps {
  router: InjectedRouter;
}

const VPPSetupContent = ({ router }: IVppSetupContentProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = useState(false);

  const uploadToken = async (data: FileList | null) => {
    setIsUploading(true);
    const token = data?.[0];
    if (!token) {
      setIsUploading(false);
      renderFlash("error", "No token selected.");
      return;
    }

    try {
      // TODO: API integration
      // await mdmAppleBmAPI.uploadToken(token);
      renderFlash(
        "success",
        "Volume Purchasing Program (VPP) integration enabled successfully."
      );
      router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT);
    } catch (e) {
      // TODO: error messages
      const msg = getErrorReason(e);
      if (msg.toLowerCase().includes("valid token")) {
        renderFlash("error", msg);
      } else {
        renderFlash("error", "Couldn't Upload. Please try again.");
      }
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className={`${baseClass}__setup-content`}>
      <p className={`${baseClass}__description`}>
        Connect Fleet to your Apple Business Manager account to enable access to
        purchased apps.
      </p>
      <VppSetupSteps extendendSteps />
      <FileUploader
        className={`${baseClass}__file-uploader ${
          isUploading ? `${baseClass}__file-uploader--loading` : ""
        }`}
        accept=".vpptoken"
        message="Content token (.vpptoken)"
        graphicName="file-vpp"
        buttonType="link"
        buttonMessage={isUploading ? "Uploading..." : "Upload"}
        onFileUpload={uploadToken}
      />
    </div>
  );
};

interface IVppDisableOrRenewContentProps {
  onDisable: () => void;
  onRenew: () => void;
}

const VPPDisableOrRenewContent = ({
  onDisable,
  onRenew,
}: IVppDisableOrRenewContentProps) => {
  return (
    <div className={`${baseClass}__disable-renew-content`}>
      <div className={`${baseClass}__info`}>
        <DataSet title="Organization name" value={"test org"} />
        <DataSet title="Location" value={"test location"} />
        <DataSet title="Renew date" value={"September 19, 2024"} />
        {/* <p>{readableDate(mdmAppleBm.renew_date)}</p> */}
      </div>
      <div className={`${baseClass}__button-wrap`}>
        <Button variant="inverse" onClick={onDisable}>
          Disable
        </Button>
        <Button variant="brand" onClick={onRenew}>
          Renew token
        </Button>
      </div>
    </div>
  );
};

interface IVppSetupPageProps {
  router: InjectedRouter;
}

const VppSetupPage = ({ router }: IVppSetupPageProps) => {
  const [showDisableModal, setShowDisableModal] = useState(false);
  const [showRenewModal, setShowRenewModal] = useState(false);

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to automatic enrollment"
          path={PATHS.ADMIN_INTEGRATIONS_VPP}
          className={`${baseClass}__back-to-vpp`}
        />
        <h1>Volume Purchasing Program (VPP)</h1>
        {false ? (
          <VPPSetupContent router={router} />
        ) : (
          <VPPDisableOrRenewContent
            onDisable={() => setShowDisableModal(true)}
            onRenew={() => setShowRenewModal(true)}
          />
        )}
      </>
      {showDisableModal && (
        <DisableVppModal onExit={() => setShowDisableModal(false)} />
      )}
      {showRenewModal && (
        <RenewVppTokenModal onExit={() => setShowRenewModal(false)} />
      )}
    </MainContent>
  );
};

export default VppSetupPage;
