import React, { useContext, useState, useCallback } from "react";
import { InjectedRouter } from "react-router";
import { isAxiosError } from "axios";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { getErrorReason } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink/BackLink";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

const baseClass = "scep-page";

interface ISetCertificateOptions {
  enable: boolean;
  successMessage: string;
  errorMessage: string;
  router: InjectedRouter;
  ndesUrl?: string;
  ndesUsername?: string;
  ndesPassword?: string;
}

const useSetCertificate = ({
  enable,
  successMessage,
  errorMessage,
  router,
  ndesUrl,
  ndesUsername,
  ndesPassword,
}: ISetCertificateOptions) => {
  const { setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [isUploading, setIsUploading] = useState(false);

  const onSetupSuccess = useCallback(() => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  }, [router]);

  const onFileUpload = useCallback(
    async (files: FileList | null) => {
      if (!files?.length) {
        renderFlash("error", "No file selected");
        return;
      }
      setIsUploading(true);
      try {
        // TODO: Replace with correct API call
        // await mdmAppleApi.uploadApplePushCertificate(files[0]);
        renderFlash("success", "MDM turned on successfully.");
        onSetupSuccess();
      } catch (e) {
        const msg = getErrorReason(e);
        if (
          msg.toLowerCase().includes("invalid certificate") ||
          msg.toLowerCase().includes("required private key")
        ) {
          renderFlash("error", msg);
        } else {
          renderFlash("error", "Couldn’t connect. Please try again.");
        }
        setIsUploading(false);
      }
    },
    [renderFlash, onSetupSuccess]
  );

  const turnOnWindowsMdm = async () => {
    try {
      const updatedConfig = await configAPI.updateMDMConfig(
        {
          windows_enabled_and_configured: enable,
        },
        true
      );
      setConfig(updatedConfig);
      renderFlash("success", successMessage);
    } catch (e) {
      let msg = errorMessage;
      if (enable && isAxiosError(e) && e.response?.status === 422) {
        msg =
          getErrorReason(e, {
            nameEquals: "mdm.windows_enabled_and_configured",
          }) || msg;
      }
      renderFlash("error", msg);
    } finally {
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    }
  };

  return turnOnWindowsMdm;
};

interface IWindowsMdmOnContentProps {
  router: InjectedRouter;
  onFileUpload: () => void;
  isUploading: boolean;
  formData: any; // TODO
  onInputChange: ({ name, value }: IFormField) => void;
}

const WindowsMdmOnContent = ({
  router,
  onFileUpload,
  isUploading,
  formData,
  onInputChange,
}: IWindowsMdmOnContentProps) => {
  const turnOnWindowsMdm = useSetCertificate({
    enable: true,
    successMessage: "Windows MDM turned on (servers excluded).",
    errorMessage: "Unable to turn on Windows MDM. Please try again.",
    router,
  });

  return (
    <>
      <h1>SCEP</h1>
      <p>
        Add a SCEP connection to enable Fleet to get SCEP certificates from your
        custom SCEP server and install them on macOS hosts.
      </p>
      <p>
        Fleet currently supports Microsoft&apos;s Network Device Enrollment
        Service (NDES) as a custom SCEP server.
      </p>
      <div>
        <ol className={`${baseClass}__setup-instructions-list`}>
          <li>
            <span>1. </span>
            <span>Configure your NDES admin account using the form below:</span>
            <div className={`${baseClass}__url-inputs-wrapper`}>
              <InputField
                inputWrapperClass={`${baseClass}__url-input`}
                label="URL"
                name="url"
                tooltip="URL of the page to use to retrieve the SCEP challenge"
                value={formData.url}
                onInputChange={onInputChange}
                placeholder="https://url.example.com"
                enableCopy
              />
              <InputField
                inputWrapperClass={`${baseClass}__url-input`}
                label="Username"
                name="username"
                tooltip="The username in the down-level logon name format required to log in to the SCEP Admin page"
                value={formData.username}
                onInputChange={onInputChange}
                placeholder="NDES admin username"
                enableCopy
              />
              <InputField
                inputWrapperClass={`${baseClass}__url-input`}
                label="Password"
                name="password"
                tooltip="The password to use to log in to the SCEP Admin page"
                value={formData.password}
                onInputChange={onInputChange}
                placeholder="NDES admin password"
              />
            </div>
          </li>
          <li>
            <span>2. </span>
            <span>
              Follow instructions to get your signing certificate from NDES{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/setup-ndes"
                text="here"
                newTab
              />
            </span>
          </li>
          <li>
            <span>3. </span>
            <span>Upload your certificate (.pfx file) below.</span>
            <br />
            <FileUploader
              className={`${baseClass}__file-uploader ${
                isUploading ? `${baseClass}__file-uploader--loading` : ""
              }`}
              accept=".pfx"
              buttonMessage={isUploading ? "Uploading..." : "Upload"}
              buttonType="link"
              disabled={isUploading}
              graphicName="file-pem"
              message="APNs certificate (.pem)"
              onFileUpload={onFileUpload}
            />
          </li>
        </ol>
      </div>
    </>
  );
};

interface IWindowsMdmOffContentProps {
  router: InjectedRouter;
}

const WindowsMdmOffContent = ({ router }: IWindowsMdmOffContentProps) => {
  const turnOffWindowsMdm = useSetCertificate({
    enable: false,
    successMessage: "Windows MDM turned off.",
    errorMessage: "Unable to turn off Windows MDM. Please try again.",
    router,
  });

  return (
    <>
      <h1>Turn off Windows MDM</h1>
      <p>
        MDM will no longer be turned on for Windows hosts that enroll to Fleet.
      </p>
      <p>Hosts with MDM already turned on MDM will not have MDM removed.</p>
      <Button onClick={turnOffWindowsMdm}>Turn off MDM</Button>
    </>
  );
};

interface IScepPageProps {
  router: InjectedRouter;
  onFileUpload: () => void;
  isUploading: boolean;
}

interface INdesFormData {
  url: string;
  username: string;
  password: string;
}

export interface IFormField {
  name: string;
  value: string;
}

const ScepPage = ({ router, onFileUpload, isUploading }: IScepPageProps) => {
  const { config } = useContext(AppContext);

  const ndesInfoReturnedFromApi = {
    url: "",
    username: "",
    password: "",
  };

  const {
    url: ndesUrl,
    username: ndesUsername,
    password: ndesPassword,
  } = ndesInfoReturnedFromApi;

  const [formData, setFormData] = useState<INdesFormData>({
    url: ndesUrl || "",
    username: ndesUsername || "",
    password: ndesPassword || "",
  });

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const isScepCertificateUploaded = false;
  // config?.mdm?.windows_enabled_and_configured ?? false; // TODO

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        {isScepCertificateUploaded ? (
          <WindowsMdmOffContent router={router} />
        ) : (
          <WindowsMdmOnContent
            router={router}
            onFileUpload={onFileUpload}
            isUploading={isUploading}
            formData={formData}
            onInputChange={onInputChange}
          />
        )}
      </>
    </MainContent>
  );
};

export default ScepPage;
