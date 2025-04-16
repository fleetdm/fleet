import React, { useContext, useState } from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import { AppContext } from "context/app";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "conditional-access-modal";

export interface IConditionalAccessFormData {
  enabled: boolean;
}

interface IConditionalAccessModal {
  onExit: () => void;
  onSubmit: (data: IConditionalAccessFormData) => void;
  configured: boolean;
  enabled: boolean;
  isUpdating: boolean;
  gitOpsModeEnabled?: boolean;
}

const ConditionalAccessModal = ({
  onExit,
  onSubmit,
  configured,
  enabled,
  isUpdating,
  gitOpsModeEnabled = false,
}: IConditionalAccessModal) => {
  const [formData, setFormData] = useState<IConditionalAccessFormData>({
    enabled,
  });

  const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);
  const isAdmin = isGlobalAdmin || isTeamAdmin;

  const onChangeEnabled = () => {
    // no validation needed, just a flag
    setFormData({ ...formData, enabled: !formData.enabled });
  };

  const handleSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    // no validation needed, just a flag
    onSubmit(formData);
  };

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
        <p>
          If enabled, single sign-on will be blocked for end users whose hosts
          fail any policies. {learnMoreLink}
        </p>
        <form onSubmit={handleSubmit} autoComplete="off">
          <Slider
            value={formData.enabled}
            onChange={onChangeEnabled}
            inactiveText="Disabled"
            activeText="Enabled"
            disabled={gitOpsModeEnabled || !isAdmin}
          />
          <GitOpsModeTooltipWrapper
            tipOffset={-8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={disableChildren || !isAdmin}
                className="button-wrap"
                isLoading={isUpdating}
              >
                Save
              </Button>
            )}
          />
        </form>
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
