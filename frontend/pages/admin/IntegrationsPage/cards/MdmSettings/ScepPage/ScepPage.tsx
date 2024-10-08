import React, { useContext, useState } from "react";
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
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import TooltipWrapper from "components/TooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import { SCEP_SERVER_TIP_CONTENT } from "../components/ScepSection/ScepSection";

const baseClass = "scep-page";

const BAD_SCEP_URL_ERROR = "Invalid SCEP URL. Please correct and try again.";
const BAD_CREDENTIALS_ERROR =
  "Invalid admin URL or credentials. Please correct and try again.";
const CACHE_ERROR =
  "The NDES password cache is full. Please increase the number of cached passwords in NDES and try again.";
const DEFAULT_ERROR =
  "Something went wrong updating your SCEP server. Please try again.";

interface ITurnOnMdmMessageProps {
  router: InjectedRouter;
}

const TurnOnMdmMessage = ({ router }: ITurnOnMdmMessageProps) => {
  return (
    <div className={`${baseClass}__turn-on-mdm-message`}>
      <h2>Turn on Apple MDM</h2>
      {/* TODO: Confirm wording missed Figma spec */}
      <p>To help your end users connect to Wi-Fi, first turn on Apple MDM.</p>
      <Button
        variant="brand"
        onClick={() => {
          router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
        }}
      >
        Turn on
      </Button>
    </div>
  );
};

interface IScepCertificateContentProps {
  router: InjectedRouter;
  onFormSubmit: (evt: React.MouseEvent<HTMLFormElement>) => Promise<void>;
  formData: any; // TODO
  onInputChange: ({ name, value }: IFormField) => void;
  config: IConfig | null;
  isPremiumTier: boolean;
  isLoading: boolean;
  isSaving: boolean;
  showDataError: boolean;
}

const ScepCertificateContent = ({
  router,
  onFormSubmit,
  formData,
  onInputChange,
  config,
  isPremiumTier,
  isLoading,
  isSaving,
  showDataError,
}: IScepCertificateContentProps) => {
  if (!isPremiumTier) {
    return <PremiumFeatureMessage />;
  }

  if (!config?.mdm.enabled_and_configured) {
    return <TurnOnMdmMessage router={router} />;
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
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="SCEP URL"
                  name="scepUrl"
                  tooltip="The URL used by client devices to request and retrieve certificates."
                  value={formData.url}
                  onInputChange={onInputChange}
                  placeholder="https://example.com/certsrv/mscep/mscep.dll"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="Admin URL"
                  name="adminUrl"
                  tooltip="The admin interface for managing the SCEP service and viewing configuration details."
                  value={formData.url}
                  onInputChange={onInputChange}
                  placeholder="https://example.com/certsrv/mscep_admin/"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="Username"
                  name="username"
                  tooltip="The username in the down-level logon name format required to log in to the SCEP admin page."
                  value={formData.username}
                  onInputChange={onInputChange}
                  placeholder="username@example.microsoft.com"
                />
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="Password"
                  name="password"
                  tooltip="The password to use to log in to the SCEP admin page."
                  value={formData.password}
                  onInputChange={onInputChange}
                  placeholder="• • • • • • • • • • • •"
                />
                <Button
                  type="submit"
                  variant="brand"
                  className="button-wrap"
                  isLoading={isSaving}
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
  url: string;
  adminUrl: string;
  username: string;
  password: string;
}

export interface IFormField {
  name: string;
  value: string;
}

const ScepPage = ({ router }: IScepPageProps) => {
  const { config, isPremiumTier, setConfig } = useContext(AppContext);

  const { renderFlash } = useContext(NotificationContext);

  const isMdmEnabled = !!config?.mdm.enabled_and_configured;

  const [isUpdatingNdesScepProxy, setIsUpdatingNdesScepProxy] = useState(false);

  const ndesInfoReturnedFromApi: IScepInfo = {
    url: "",
    admin_url: "",
    username: "",
    password: "",
  };

  const {
    url,
    admin_url: adminUrl,
    username,
    password,
  } = ndesInfoReturnedFromApi;

  const [formData, setFormData] = useState<INdesFormData>({
    url: "",
    adminUrl: "",
    username: "",
    password: "",
  });

  const {
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
    isError: isErrorAppConfig,
    error: errorAppConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      if (data.integrations.ndes_scep_proxy) {
        setFormData({
          url: data.integrations.ndes_scep_proxy.url || "",
          adminUrl: data.integrations.ndes_scep_proxy.admin_url || "",
          username: data.integrations.ndes_scep_proxy.username || "",
          password: data.integrations.ndes_scep_proxy.password || "",
        });
      }
    },
  });

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const onFormSubmit = async (evt: React.MouseEvent<HTMLFormElement>) => {
    setIsUpdatingNdesScepProxy(true);

    evt.preventDefault();

    const isRemovingNdesScepProxy =
      formData.url === "" &&
      formData.adminUrl === "" &&
      username === "" &&
      password === "";

    // Format for API
    const formDataToSubmit = isRemovingNdesScepProxy
      ? null // Send null if no fields are set
      : [
          {
            url: formData.url,
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
            onInputChange={onInputChange}
            config={config}
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
