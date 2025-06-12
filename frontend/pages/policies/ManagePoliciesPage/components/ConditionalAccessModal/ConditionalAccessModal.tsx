import React, { useContext, useRef, useState } from "react";

import {
  FLEET_WEBSITE_URL,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import { AppContext } from "context/app";
import { IPaginatedListHandle } from "components/PaginatedList";
import PoliciesPaginatedList, {
  IFormPolicy,
} from "../PoliciesPaginatedList/PoliciesPaginatedList";

const baseClass = "conditional-access-modal";

export interface IConditionalAccessFormData {
  enabled: boolean;
  changedPolicies: IFormPolicy[];
}

interface IConditionalAccessModal {
  onExit: () => void;
  onSubmit: (data: IConditionalAccessFormData) => void;
  configured: boolean;
  enabled: boolean;
  isUpdating: boolean;
  gitOpsModeEnabled?: boolean;
  teamId: number;
}

const ConditionalAccessModal = ({
  onExit,
  onSubmit,
  configured,
  enabled,
  isUpdating,
  gitOpsModeEnabled = false,
  teamId,
}: IConditionalAccessModal) => {
  const [formData, setFormData] = useState<IConditionalAccessFormData>({
    enabled,
    changedPolicies: [],
  });

  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);
  const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);
  const isAdmin = isGlobalAdmin || isTeamAdmin;

  const onChangeEnabled = () => {
    // no validation needed, just a flag
    setFormData({ ...formData, enabled: !formData.enabled });
  };

  const handleSubmit = () => {
    if (paginatedListRef.current) {
      const changedPolicies = paginatedListRef.current.getDirtyItems();
      onSubmit({ ...formData, changedPolicies });
    }
  };

  const getPolicyDisabled = (policy: IFormPolicy) =>
    !policy.platform.includes("darwin");

  const getPolicyTooltipContent = (policy: IFormPolicy) =>
    !policy.platform.includes("darwin") ? "Policy does not target macOS" : null;

  const learnMoreLink = (
    <CustomLink
      text="Learn more"
      url={`${LEARN_MORE_ABOUT_BASE_LINK}/conditional-access`}
      newTab
    />
  );

  const renderConfigured = () => {
    return (
      <>
        <div className="form">
          <span className="header">
            <Slider
              value={formData.enabled}
              onChange={onChangeEnabled}
              inactiveText="Disabled"
              activeText="Enabled"
              disabled={gitOpsModeEnabled || !isAdmin}
            />
            <CustomLink
              text="Preview end user experience"
              newTab
              multiline={false}
              url={`${FLEET_WEBSITE_URL}/microsoft-compliance-partner/remediate`}
            />
          </span>
          <PoliciesPaginatedList
            ref={paginatedListRef}
            isSelected="conditional_access_enabled"
            getPolicyDisabled={getPolicyDisabled}
            getPolicyTooltipContent={getPolicyTooltipContent}
            onToggleItem={(item: IFormPolicy) => {
              item.conditional_access_enabled = !item.conditional_access_enabled;
              return item;
            }}
            helpText={
              <>
                Single sign-on will be blocked for end users whose hosts fail
                any of these policies.{" "}
                <CustomLink
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/conditional-access`}
                  text="Learn more"
                  newTab
                  disableKeyboardNavigation={!formData.enabled}
                />
              </>
            }
            isUpdating={isUpdating}
            onSubmit={handleSubmit}
            onCancel={onExit}
            teamId={teamId}
            disableList={!formData.enabled}
          />
        </div>
      </>
    );
  };

  const renderNotConfigured = () => (
    <>
      To block single sign-on from hosts failing policies, you must first
      connect Fleet to Microsoft Entra.
      <br />
      <br />
      This can be configured in <b>Settings</b> &gt; <b>Integrations</b> &gt;{" "}
      <b>Conditional access</b>.
      <br />
      <br />
      {learnMoreLink}
      <div className="modal-cta-wrap">
        <Button onClick={onExit}>Done</Button>
      </div>
    </>
  );
  return (
    <Modal
      className={baseClass}
      title="Conditional access"
      onExit={onExit}
      width="large"
    >
      {configured ? renderConfigured() : renderNotConfigured()}
    </Modal>
  );
};

export default ConditionalAccessModal;
