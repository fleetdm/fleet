import React, { useContext } from "react";

import { AppContext } from "context/app";
import { IIntegrationType } from "interfaces/integration";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

import JiraPreview from "../../../../../assets/images/jira-vuln-software-preview-400x517@2x.png";
import ZendeskPreview from "../../../../../assets/images/zendesk-vuln-software-preview-400x455@2x.png";
import JiraPreviewPremium from "../../../../../assets/images/jira-vuln-software-preview-premium-400x517@2x.png";
import ZendeskPreviewPremium from "../../../../../assets/images/zendesk-vuln-software-preview-premium-400x455@2x.png";

const baseClass = "preview-ticket-modal";

interface IPreviewTicketModalProps {
  onCancel: () => void;
  integrationType: IIntegrationType;
}

const PreviewTicketModal = ({
  onCancel,
  integrationType,
}: IPreviewTicketModalProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const screenshot =
    integrationType === "jira" ? (
      <img
        src={isPremiumTier ? JiraPreviewPremium : JiraPreview}
        alt="Jira ticket"
        className={`${baseClass}__jira-screenshot`}
      />
    ) : (
      <img
        src={isPremiumTier ? ZendeskPreviewPremium : ZendeskPreview}
        alt="Zendesk ticket"
        className={`${baseClass}__zendesk-screenshot`}
      />
    );

  return (
    <Modal
      title="Example ticket"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="large"
    >
      <>
        <p className="automations-learn-more">
          Want to learn more about how automations in Fleet work?{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/automations"
            text="Check out the Fleet documentation"
            newTab
          />
        </p>
        <div className={`${baseClass}__example`}>{screenshot}</div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default PreviewTicketModal;
