import React, { useState } from "react";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import CustomLink from "components/CustomLink";
import PageDescription from "components/PageDescription";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import { notify } from "components/ToastNotification";
import useGitOpsMode from "hooks/useGitOpsMode";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import SettingsSection from "pages/admin/components/SettingsSection";
import { IEulaMetadataResponse } from "services/entities/mdm";

import DeleteEulaModal from "./components/DeleteEulaModal/DeleteEulaModal";
import EulaUploader from "./components/EulaUploader/EulaUploader";
import ExampleWindowsEulaModal from "./components/ExampleWindowsEulaModal";
import UploadedEulaView from "./components/UploadedEulaView/UploadedEulaView";
import { EULA_PLATFORM_CONFIG, EULA_PLATFORMS, EulaPlatform } from "./helpers";

const baseClass = "eula-section";

export interface IPlatformEula {
  isAvailable: boolean;
  metadata?: IEulaMetadataResponse;
  onChange: () => void;
}

interface IEulaSectionProps {
  eulas: Record<EulaPlatform, IPlatformEula>;
}

const EulaSection = ({ eulas }: IEulaSectionProps) => {
  const { gitOpsModeEnabled } = useGitOpsMode();
  const [selectedTabIndex, setSelectedTabIndex] = useState(() =>
    Math.max(
      0,
      EULA_PLATFORMS.findIndex((platform) => eulas[platform].isAvailable)
    )
  );
  const [platformToDelete, setPlatformToDelete] = useState<EulaPlatform | null>(
    null
  );
  const [isDeleting, setIsDeleting] = useState(false);
  const [showExampleModal, setShowExampleModal] = useState(false);

  const onDeleteEula = async () => {
    if (!platformToDelete || isDeleting || gitOpsModeEnabled) return;
    const eula = eulas[platformToDelete];
    if (!eula.metadata) return;

    setIsDeleting(true);
    try {
      await EULA_PLATFORM_CONFIG[platformToDelete].remove(eula.metadata.token);
      notify.success("Successfully deleted.");
    } catch (e) {
      notify.error("Couldn’t delete. Please try again.", { response: e });
    } finally {
      setIsDeleting(false);
      setPlatformToDelete(null);
      eula.onChange();
    }
  };

  const renderPlatformPanel = (platform: EulaPlatform) => {
    const eula = eulas[platform];
    if (!eula.isAvailable) {
      return (
        <p className={`${baseClass}__unavailable`}>
          {EULA_PLATFORM_CONFIG[platform].unavailableMessage}
        </p>
      );
    }
    if (!eula.metadata) {
      return <EulaUploader platform={platform} onUpload={eula.onChange} />;
    }
    return (
      <UploadedEulaView
        platform={platform}
        eulaMetadata={eula.metadata}
        onDelete={() => setPlatformToDelete(platform)}
        onShowExample={() => setShowExampleModal(true)}
      />
    );
  };

  return (
    <SettingsSection
      className={baseClass}
      title="End user license agreement (EULA)"
      id="end-user-license-agreement"
    >
      <PageDescription
        variant="right-panel"
        content={
          <>
            Require end users to agree to terms during initial setup.{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/end-user-license-agreement"
              text="Learn more"
              newTab
            />
          </>
        }
      />
      <div className={`${baseClass}__content`}>
        <TabNav secondary>
          <Tabs selectedIndex={selectedTabIndex} onSelect={setSelectedTabIndex}>
            <TabList>
              {EULA_PLATFORMS.map((platform) => (
                <Tab key={platform}>
                  <TabText>{PLATFORM_DISPLAY_NAMES[platform]}</TabText>
                </Tab>
              ))}
            </TabList>
            {EULA_PLATFORMS.map((platform) => (
              <TabPanel key={platform}>
                {renderPlatformPanel(platform)}
              </TabPanel>
            ))}
          </Tabs>
        </TabNav>
      </div>
      {platformToDelete && (
        <DeleteEulaModal
          platform={platformToDelete}
          isDeleting={isDeleting}
          onDelete={onDeleteEula}
          onCancel={() => setPlatformToDelete(null)}
        />
      )}
      {showExampleModal && (
        <ExampleWindowsEulaModal onExit={() => setShowExampleModal(false)} />
      )}
    </SettingsSection>
  );
};

export default EulaSection;
