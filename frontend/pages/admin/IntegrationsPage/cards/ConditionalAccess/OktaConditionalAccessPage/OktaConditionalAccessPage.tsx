import React, { useContext, useEffect, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import Editor from "components/Editor";
import MainContent from "components/MainContent";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { IConfig, isOktaConditionalAccessConfigured } from "interfaces/config";
import { getErrorReason } from "interfaces/errors";
import PATHS from "router/paths";
import conditionalAccessAPI, {
  IConditionalAccessWindowsProfilesResponse,
} from "services/entities/conditional_access";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import OktaConditionalAccessModal from "../components/OktaConditionalAccessModal";

const baseClass = "okta-conditional-access-page";

interface IProfileEditorProps {
  label: string;
  name: string;
  value: string;
}

const ProfileEditor = ({ label, name, value }: IProfileEditorProps) => (
  <Editor
    label={label}
    ariaLabel={label.toLowerCase()}
    name={name}
    mode="xml"
    value={value}
    readOnly
    enableCopy
    copyButtonLocation="label"
    wrapEnabled
    maxLines={10}
  />
);

// Each tab fetches its own profiles when opened, so an error that only
// affects one platform doesn't show up while viewing the other.
const MacOSProfileTab = () => {
  const { data: appleProfile = "" } = useQuery<string, Error>(
    ["appleProfile"],
    conditionalAccessAPI.getIdpAppleProfile,
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      onError: (e: unknown) => {
        // When responseType is "text", error responses come back as JSON strings
        // that need to be parsed manually
        const data = (e as { data?: unknown } | undefined)?.data;
        let errorReason = "";
        try {
          if (data && typeof data === "string") {
            const parsedError = JSON.parse(data);
            errorReason = parsedError.errors?.[0]?.reason || "";
          } else {
            errorReason = getErrorReason(e);
          }
        } catch {
          errorReason = getErrorReason(e);
        }

        const message = errorReason
          ? `Failed to load Apple profile: ${errorReason}`
          : "Failed to load Apple profile.";
        notify.error(message, { response: e });
      },
    }
  );

  return (
    <div className={`${baseClass}__profiles`}>
      <ProfileEditor
        label="Profile"
        name="macos-profile"
        value={appleProfile}
      />
    </div>
  );
};

const WindowsProfilesTab = () => {
  const { data: windowsProfiles } = useQuery<
    IConditionalAccessWindowsProfilesResponse,
    Error
  >(["windowsProfiles"], conditionalAccessAPI.getIdpWindowsProfiles, {
    ...DEFAULT_USE_QUERY_OPTIONS,
    onError: (e: unknown) => {
      const reason = getErrorReason(e);
      notify.error(
        reason
          ? `Failed to load Windows profiles: ${reason}`
          : "Failed to load Windows profiles.",
        { response: e }
      );
    },
  });

  return (
    <div className={`${baseClass}__profiles`}>
      <ProfileEditor
        label="Certificate profile"
        name="windows-certificate-profile"
        value={windowsProfiles?.scep_profile ?? ""}
      />
      <ProfileEditor
        label="Browser profile"
        name="windows-browser-profile"
        value={windowsProfiles?.browser_profile ?? ""}
      />
    </div>
  );
};

interface IOktaConditionalAccessPageProps {
  router: InjectedRouter;
}

const OktaConditionalAccessPage = ({
  router,
}: IOktaConditionalAccessPageProps) => {
  const { config, setConfig } = useContext(AppContext);
  const [showSettingsModal, setShowSettingsModal] = useState(false);

  const oktaConfigured = isOktaConditionalAccessConfigured(config);

  useEffect(() => {
    if (config && !oktaConfigured) {
      router.replace(PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS);
    }
  }, [config, oktaConfigured, router]);

  if (!oktaConfigured) {
    return null;
  }

  const onSettingsSaved = (updatedConfig: IConfig) => {
    setShowSettingsModal(false);
    setConfig(updatedConfig);
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__header-links`}>
        <BackButton
          text="Back to conditional access"
          path={PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS}
        />
      </div>
      <h1>Okta conditional access</h1>
      <div className={`${baseClass}__content`}>
        <TabNav secondary>
          <Tabs>
            <TabList>
              <Tab>
                <TabText>macOS</TabText>
              </Tab>
              <Tab>
                <TabText>Windows</TabText>
              </Tab>
            </TabList>
            <TabPanel>
              <MacOSProfileTab />
            </TabPanel>
            <TabPanel>
              <WindowsProfilesTab />
            </TabPanel>
          </Tabs>
        </TabNav>
        <div className={`${baseClass}__actions`}>
          <Button onClick={() => setShowSettingsModal(true)}>Settings</Button>
        </div>
      </div>
      {showSettingsModal && (
        <OktaConditionalAccessModal
          onCancel={() => setShowSettingsModal(false)}
          onSuccess={onSettingsSaved}
        />
      )}
    </MainContent>
  );
};

export default OktaConditionalAccessPage;
