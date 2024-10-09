import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import configAPI from "services/entities/config";
import { IScepInfo } from "services/entities/mdm_apple";
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
  "Invalid admin URL or credentials. Please correct and try again.";
const CACHE_ERROR =
  "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again.";
const DEFAULT_ERROR =
  "Something went wrong updating your SCEP server. Please try again.";

interface IScepCertificateContentProps {
  router: InjectedRouter;
  onFormSubmit: (evt: React.MouseEvent<HTMLFormElement>) => Promise<void>;
  formData: INdesFormData; // TODO
  formErrors: INdesFormErrors;
  onInputChange: ({ name, value }: IFormField) => void;
  config: IConfig | null;
  isPremiumTier: boolean;
  isLoading: boolean;
  isSaving: boolean;
  isSavingDisabled: boolean;
  showDataError: boolean;
}

const ScepCertificateContent = ({
  router,
  onFormSubmit,
  formData,
  formErrors,
  onInputChange,
  config,
  isPremiumTier,
  isLoading,
  isSaving,
  isSavingDisabled,
  showDataError,
}: IScepCertificateContentProps) => {
  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
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

  if (isLoading) {
    return <Spinner />;
  }

  // TODO: error UI
  if (showDataError) {
    return (
      <div>
        <DataError />
      </div>
    );
  }

  const allEmptyValues =
    !formData.scepUrl &&
    !formData.adminUrl &&
    !formData.username &&
    !formData.password;

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
            {/* TODO: confirm URL */}
            <>
              Connect to your Network Device Enrollment Service (
              <CustomLink
                url="https://www.fleetdm.com/learn-more-about/ndes"
                text="NDES"
              />
              ) admin account:
            </>
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
                  parseTarget
                  onChange={onInputChange}
                  error={!!formErrors.scepUrl && formErrors.scepUrl}
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
                  parseTarget
                  onChange={onInputChange}
                  error={!!formErrors.adminUrl && formErrors.adminUrl}
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
                  parseTarget
                  onChange={onInputChange}
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
                  parseTarget
                  onChange={onInputChange}
                  placeholder="••••••••"
                  blockAutoComplete
                />
                <Button
                  type="submit"
                  variant="brand"
                  className="button-wrap"
                  isLoading={isSaving}
                  disabled={isSavingDisabled}
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
  password: null;
}

interface INdesFormErrors {
  scepUrl?: string | null;
  adminUrl?: string | null;
}

export interface IFormField {
  name: string;
  value: string;
}

const ScepPage = ({ router }: IScepPageProps) => {
  const { config, isPremiumTier, setConfig } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const isMdmEnabled = !!config?.mdm.enabled_and_configured;

  const ndesInfoReturnedFromApi: IScepInfo = {
    url: "",
    admin_url: "",
    username: "",
    password: null,
  };

  const {
    url: scepUrl,
    admin_url: adminUrl,
    username,
    password,
  } = ndesInfoReturnedFromApi;

  const [formData, setFormData] = useState<INdesFormData>({
    scepUrl: "",
    adminUrl: "",
    username: "",
    password: null,
  });
  const [formErrors, setFormErrors] = useState<INdesFormErrors>({});
  const [isSavingDisabled, setIsSavingDisabled] = useState(true);
  const [isUpdatingNdesScepProxy, setIsUpdatingNdesScepProxy] = useState(false);

  const {
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
    isError: isErrorAppConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      if (data.integrations.ndes_scep_proxy) {
        setFormData({
          scepUrl: data.integrations.ndes_scep_proxy.url || "",
          adminUrl: data.integrations.ndes_scep_proxy.admin_url || "",
          username: data.integrations.ndes_scep_proxy.username || "",
          password: null,
        });
      }
    },
  });

  useEffect(() => {
    const allFieldsEmpty =
      formData.scepUrl === "" &&
      formData.adminUrl === "" &&
      formData.username === "" &&
      (formData.password === "" || formData.password === null);

    const allFieldsPresent =
      formData.scepUrl !== "" &&
      formData.adminUrl !== "" &&
      formData.username !== "" &&
      formData.password !== "";

    const isPasswordNull = formData.password === null;

    if (!allFieldsEmpty && !allFieldsPresent) {
      setIsSavingDisabled(true);
    } else setIsSavingDisabled(false);
  }, [formData]);

  const onInputChange = ({ name, value }: IFormField) => {
    console.log("setFormData");
    setFormData({ ...formData, [name]: value });
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

    if (!scepUrlValid && !adminUrlValid) {
      return;
    }

    setIsUpdatingNdesScepProxy(true);

    const isRemovingNdesScepProxy =
      formData.scepUrl === "" &&
      formData.adminUrl === "" &&
      username === "" &&
      password === "";

    // Format for API
    const formDataToSubmit = isRemovingNdesScepProxy
      ? null // Send null if no fields are set
      : [
          {
            url: formData.scepUrl,
            admin_url: formData.adminUrl,
            username: formData.username,
            password: formData.password || null,
          },
        ];

    // Update integrations.ndes_scep_proxy only
    const destination = {
      ndes_scep_proxy: formDataToSubmit,
    };

    try {
      await configAPI.update({ integrations: destination });
      renderFlash("success", "Successfully added your SCEP server.");
      refetchConfig();
    } catch (error) {
      console.error(error);
      if (getErrorReason(error).includes("TODO")) {
        renderFlash("error", BAD_SCEP_URL_ERROR);
      } else if (getErrorReason(error).includes("TODO")) {
        renderFlash("error", BAD_CREDENTIALS_ERROR);
      } else if (getErrorReason(error).includes("TODO")) {
        renderFlash("error", CACHE_ERROR);
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
            config={config}
            isPremiumTier={isPremiumTier || false}
            isLoading={isLoadingAppConfig}
            isSaving={isUpdatingNdesScepProxy}
            isSavingDisabled={isSavingDisabled}
            showDataError={isErrorAppConfig}
          />
        </div>
      </>
    </MainContent>
  );
};

export default ScepPage;
