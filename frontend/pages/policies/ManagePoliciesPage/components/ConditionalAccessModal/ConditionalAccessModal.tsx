import React, { useContext, useRef, useState } from "react";

import {
  FLEET_WEBSITE_URL,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import { isOktaConditionalAccessConfigured } from "interfaces/config";

import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
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
  providerText: string;
}

const ConditionalAccessModal = ({
  onExit,
  onSubmit,
  configured,
  enabled,
  isUpdating,
  gitOpsModeEnabled = false,
  teamId,
  providerText,
}: IConditionalAccessModal) => {
  const [formData, setFormData] = useState<IConditionalAccessFormData>({
    enabled,
    changedPolicies: [],
  });

  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);
  const { isGlobalAdmin, isTeamAdmin, config } = useContext(AppContext);
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

  const renderItemRow = (
    item: IFormPolicy,
    onChange: (item: IFormPolicy) => void
  ) => {
    const shouldShowCheckbox =
      item.conditional_access_enabled &&
      // currently redundant as only darwin-targeting policies are enabled in this list
      item.platform.includes("darwin") &&
      isOktaConditionalAccessConfigured(config) &&
      !config?.conditional_access?.bypass_disabled;

    if (!shouldShowCheckbox) {
      return null;
    }

    return (
      <span
        onClick={(e) => {
          e.stopPropagation();
        }}
      >
        <Checkbox
          value={item.conditional_access_bypass_enabled}
          onChange={() => {
            onChange({
              ...item,
              conditional_access_bypass_enabled: !item.conditional_access_bypass_enabled,
            });
          }}
        >
          <TooltipWrapper tipContent={
              <>
              Allows end users to bypass conditional access for a single login if they are unable to resolve the failing policy.
              <br/>
              <br/>
              <em>This experimental setting will be removed in Fleet 4.83, and only non-critical policies will allow bypass. For a seamless upgrade, please avoid enabling bypass for policies marked critical.</em>
              </>
            }>
            End users can bypass
          </TooltipWrapper>
        </Checkbox>
      </span>
    );
  };

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
            renderItemRow={renderItemRow}
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
      connect Fleet to {providerText}.
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
