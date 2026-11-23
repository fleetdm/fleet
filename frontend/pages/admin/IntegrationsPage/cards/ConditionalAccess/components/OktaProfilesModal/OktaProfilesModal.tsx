import React from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import Button from "components/buttons/Button";
import Editor from "components/Editor";
import Modal from "components/Modal";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import conditionalAccessAPI, {
  IConditionalAccessWindowsProfilesResponse,
} from "services/entities/conditional_access";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { getConditionalAccessErrorMessage } from "../../helpers";

const baseClass = "okta-profiles-modal";

interface IProfileEditorProps {
  label: string;
  name: string;
  value: string;
  helpText?: string;
}

const ProfileEditor = ({
  label,
  name,
  value,
  helpText,
}: IProfileEditorProps) => (
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
    helpText={helpText}
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

        notify.error(
          getConditionalAccessErrorMessage("load Apple profile", errorReason),
          { response: e }
        );
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
      notify.error(
        getConditionalAccessErrorMessage(
          "load Windows profiles",
          getErrorReason(e)
        ),
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

const PLATFORM_TABS = [
  { name: "macOS", Panel: MacOSProfileTab },
  { name: "Windows", Panel: WindowsProfilesTab },
];

interface IOktaProfilesModalProps {
  onExit: () => void;
}

const OktaProfilesModal = ({ onExit }: IOktaProfilesModalProps) => (
  <Modal
    title="Okta conditional access profiles"
    onExit={onExit}
    className={baseClass}
    width="large"
  >
    <TabNav>
      <Tabs>
        <TabList>
          {PLATFORM_TABS.map(({ name }) => (
            // data-text reserves the bold width so the active tab doesn't shift
            <Tab key={name} data-text={name}>
              <TabText>{name}</TabText>
            </Tab>
          ))}
        </TabList>
        {PLATFORM_TABS.map(({ name, Panel }) => (
          <TabPanel key={name}>
            <Panel />
          </TabPanel>
        ))}
      </Tabs>
    </TabNav>
    <div className="modal-cta-wrap">
      <Button onClick={onExit}>Close</Button>
    </div>
  </Modal>
);

export default OktaProfilesModal;
