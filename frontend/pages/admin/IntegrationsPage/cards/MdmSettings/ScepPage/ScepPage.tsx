import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { IConfig } from "interfaces/config";
import { getErrorReason } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import MainContent from "components/MainContent/MainContent";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink/BackLink";
import CustomLink from "components/CustomLink";
import Card from "components/Card";
import validateUrl from "components/forms/validators/valid_url";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import { SCEP_SERVER_TIP_CONTENT } from "../components/ScepSection/ScepSection";

const baseClass = "scep-page";

const BAD_SCEP_URL_ERROR = "Invalid SCEP URL. Please correct and try again.";
const BAD_CREDENTIALS_ERROR =
  "Couldn't add. Admin URL or credentials are invalid.";
const CACHE_ERROR =
  "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again. By default, NDES caches 5 passwords and they expire 60 minutes after they are created.";
const INSUFFICIENT_PERMISSIONS_ERROR =
  "Couldn't add. This account doesn't have sufficient permissions. Please use the account with enroll permission.";
const SCEP_URL_TIMEOUT_ERROR =
  "Couldn't add. Request to NDES (SCEP URL) timed out. Please try again.";
const ADMIN_URL_TIMEOUT_ERROR =
  "Couldn't add. Request to NDES (admin URL) timed out. Please try again.";
const DEFAULT_ERROR =
  "Something went wrong updating your SCEP server. Please try again.";

interface IScepCertificateContentProps {
  router: InjectedRouter;
  onFormSubmit: (evt: React.MouseEvent<HTMLFormElement>) => Promise<void>;
  formData: INdesFormData;
  formErrors: INdesFormErrors;
  onInputChange: ({ name, value }: IFormField) => void;
  onBlur: (name: string, value: string) => void;
  config: IConfig | null;
  isPremiumTier: boolean;
  isLoading: boolean;
  isSaving: boolean;
  showDataError: boolean;
}

export const ScepCertificateContent = ({
  router,
  onFormSubmit,
  formData,
  formErrors,
  onInputChange,
  onBlur,
  config,
  isPremiumTier,
  isLoading,
  isSaving,
  showDataError,
}: IScepCertificateContentProps) => {
  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  if (isLoading) {
    return <Spinner />;
  }

  if (!config?.mdm.enabled_and_configured) {
    return (
      <TurnOnMdmMessage
        router={router}
        header="Turn on Apple MDM"
        info="To help your end users connect to Wi-Fi, first turn on Apple MDM."
      />
    );
  }

  // TODO: error UI
  if (showDataError) {
    return (
      <div>
        <DataError />
      </div>
    );
  }

  const disableSave =
    // all fields aren't empty
    !Object.values(formData).every((val) => val === "") &&
    // all fields aren't complete
    !Object.values(formData).every((val) => val !== "");

  return (
    <>
      <p>
        To help your end users connect to Wi-Fi you can add your{" "}
        <TooltipWrapper tipContent={SCEP_SERVER_TIP_CONTENT}>
          SCEP server
        </TooltipWrapper>
        .
      </p>
      <div>
        <ol className={`${baseClass}__steps`}>
          <li>
            <div>
              Connect to your Network Device Enrollment Service (
              <CustomLink
                url="https://www.fleetdm.com/learn-more-about/ndes"
                text="NDES"
              />
              ) admin account:
            </div>
            <Card>
              <form onSubmit={onFormSubmit} autoComplete="off">
                <InputField
                  inputWrapperClass={`${baseClass}__scep-url-input`}
                  label="SCEP URL"
                  name="scepUrl"
                  tooltip={
                    <>
                      The URL used by client devices
                      <br /> to request and retrieve certificates.
                    </>
                  }
                  value={formData.scepUrl}
                  onChange={onInputChange}
                  parseTarget
                  error={formErrors.scepUrl}
                  placeholder="https://example.com/certsrv/mscep/mscep.dll"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__admin-url-input`}
                  label="Admin URL"
                  name="adminUrl"
                  tooltip={
                    <>
                      The admin interface for managing the SCEP
                      <br /> service and viewing configuration details.
                    </>
                  }
                  value={formData.adminUrl}
                  onChange={onInputChange}
                  onBlur={(e: any) => onBlur("adminUrl", e.target.value)}
                  parseTarget
                  error={formErrors.adminUrl}
                  placeholder="https://example.com/certsrv/mscep_admin/"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__username-input`}
                  label="Username"
                  name="username"
                  tooltip={
                    <>
                      The username in the down-level logon name format
                      <br />
                      required to log in to the SCEP admin page.
                    </>
                  }
                  value={formData.username}
                  onChange={onInputChange}
                  onBlur={(e: any) => onBlur("username", e.target.value)}
                  parseTarget
                  placeholder="username@example.microsoft.com"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__password-input`}
                  label="Password"
                  name="password"
                  tooltip={
                    <>
                      The password to use to log in
                      <br />
                      to the SCEP admin page.
                    </>
                  }
                  value={formData.password || ""}
                  type="password"
                  onChange={onInputChange}
                  parseTarget
                  placeholder="••••••••"
                  blockAutoComplete
                  error={formErrors.password}
                />
                <Button
                  type="submit"
                  variant="brand"
                  className="button-wrap"
                  isLoading={isSaving}
                  disabled={disableSave}
                >
                  Save
                </Button>
              </form>
            </Card>
          </li>
          <li>
            <span>
              Now head over to{" "}
              <CustomLink
                url={PATHS.CONTROLS_CUSTOM_SETTINGS}
                text="Controls > OS Settings > Custom settings"
              />{" "}
              to configure how SCEP certificates are delivered to your hosts.{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/setup-ndes"
                text="Learn more"
                newTab
              />
            </span>
          </li>
        </ol>
      </div>
    </>
  );
};

interface IScepPageProps {
  router: InjectedRouter;
}

interface INdesFormData {
  scepUrl: string;
  adminUrl: string;
  username: string;
  password: string;
}

interface INdesFormErrors {
  scepUrl?: string | null;
  adminUrl?: string | null;
  password?: string | null;
}

export interface IFormField {
  name: string;
  value: string;
}

const ScepPage = ({ router }: IScepPageProps) => {
  const { isPremiumTier, config, setConfig } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const [formData, setFormData] = useState<INdesFormData>({
    scepUrl: config?.integrations.ndes_scep_proxy?.url || "",
    adminUrl: config?.integrations.ndes_scep_proxy?.admin_url || "",
    username: config?.integrations.ndes_scep_proxy?.username || "",
    password: config?.integrations.ndes_scep_proxy?.password || "",
  });

  const [formErrors, setFormErrors] = useState<INdesFormErrors>({});
  const [isUpdatingNdesScepProxy, setIsUpdatingNdesScepProxy] = useState(false);

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
    isError: isErrorAppConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      setConfig(data);
    },
  });

  const onInputChange = ({ name, value }: IFormField) => {
    setFormErrors((prev) => ({ ...prev, [name]: null }));
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleBlur = (name: string, value: string) => {
    // If the value of admin url or username has changed and
    // it was not originally empty, prompt user to re-enter password
    if (
      (name === "adminUrl" &&
        value !== config?.integrations.ndes_scep_proxy?.admin_url &&
        config?.integrations.ndes_scep_proxy?.admin_url !== "") ||
      (name === "username" &&
        value !== config?.integrations.ndes_scep_proxy?.username &&
        config?.integrations.ndes_scep_proxy?.username !== "")
    ) {
      setFormErrors((prev: INdesFormErrors) => ({
        ...prev,
        password:
          "Please re-enter your password due to changes in admin URL or username",
      }));
      setFormData((prev: INdesFormData) => ({ ...prev, password: "" }));
    }
  };

  const onFormSubmit = async (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const scepUrlValid = validateUrl({ url: formData.scepUrl });
    const adminUrlValid = validateUrl({ url: formData.adminUrl });
    const newFormErrors = {
      scepUrl:
        scepUrlValid || formData.scepUrl === ""
          ? undefined
          : "Must be a valid URL.",
      adminUrl:
        adminUrlValid || formData.adminUrl === ""
          ? undefined
          : "Must be a valid URL.",
    };

    setFormErrors(newFormErrors);

    // Remove when all fields set to empty
    const isRemovingNdesScepProxy = Object.values(formData).every(
      (val) => val === ""
    );

    if (!isRemovingNdesScepProxy && (!scepUrlValid || !adminUrlValid)) {
      return;
    }

    setIsUpdatingNdesScepProxy(true);

    // Format for API
    const formDataToSubmit = isRemovingNdesScepProxy
      ? null
      : {
          url: formData.scepUrl,
          admin_url: formData.adminUrl,
          username: formData.username,
          password: formData.password,
        };
    // Update integrations.ndes_scep_proxy only
    const destination = {
      ndes_scep_proxy: formDataToSubmit,
    };

    try {
      await configAPI.update({ integrations: destination });
      renderFlash(
        "success",
        `Successfully ${
          isRemovingNdesScepProxy ? "removed" : "added"
        } your SCEP server.`
      );
      refetchConfig();
    } catch (error) {
      console.error(error);
      const reason = getErrorReason(error);
      if (reason.includes("invalid admin URL or credentials")) {
        renderFlash("error", BAD_CREDENTIALS_ERROR);
      } else if (reason.includes("the password cache is full")) {
        renderFlash("error", CACHE_ERROR);
      } else if (reason.includes("does not have sufficient permissions")) {
        renderFlash("error", INSUFFICIENT_PERMISSIONS_ERROR);
      } else if (
        reason.includes(formData.scepUrl) &&
        reason.includes("context deadline exceeded")
      ) {
        renderFlash("error", SCEP_URL_TIMEOUT_ERROR);
      } else if (
        reason.includes(formData.adminUrl) &&
        reason.includes("context deadline exceeded")
      ) {
        renderFlash("error", ADMIN_URL_TIMEOUT_ERROR);
      } else if (reason.includes("invalid SCEP URL")) {
        renderFlash("error", BAD_SCEP_URL_ERROR);
      } else renderFlash("error", DEFAULT_ERROR);
    } finally {
      setIsUpdatingNdesScepProxy(false);
    }
  };

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        <div className={`${baseClass}__page-content`}>
          <div className={`${baseClass}__page-header-section`}>
            <h1>Simple Certificate Enrollment Protocol (SCEP)</h1>
          </div>

          <ScepCertificateContent
            router={router}
            onFormSubmit={onFormSubmit}
            formData={formData}
            formErrors={formErrors}
            onInputChange={onInputChange}
            onBlur={handleBlur}
            config={appConfig || null}
            isPremiumTier={isPremiumTier || false}
            isLoading={isLoadingAppConfig}
            isSaving={isUpdatingNdesScepProxy}
            showDataError={isErrorAppConfig}
          />
        </div>
      </>
    </MainContent>
  );
};

export default ScepPage;
