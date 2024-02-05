import React, { useContext } from "react";

import { AppContext } from "context/app";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

import { IIntegrationType } from "interfaces/integration";

import JiraPreview from "../../../../../../assets/images/jira-policy-automation-preview-400x419@2x.png";
import ZendeskPreview from "../../../../../../assets/images/zendesk-policy-automation-preview-400x515@2x.png";
import JiraPreviewPremium from "../../../../../../assets/images/jira-policy-automation-preview-premium-400x316@2x.png";
import ZendeskPreviewPremium from "../../../../../../assets/images/zendesk-policy-automation-preview-premium-400x483@2x.png";

const baseClass = "preview-ticket-modal";

interface IPreviewTicketModalProps {
  integrationType?: IIntegrationType;
  onCancel: () => void;
}

const PreviewTicketModal = ({
  integrationType,
  onCancel,
}: IPreviewTicketModalProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const screenshot =
    integrationType === "jira" ? (
      <img
        src={isPremiumTier ? JiraPreviewPremium : JiraPreview}
        alt="Jira example policy automation ticket"
        className={`${baseClass}__screenshot`}
      />
    ) : (
      <img
        src={isPremiumTier ? ZendeskPreviewPremium : ZendeskPreview}
        alt="Zendesk example policy automation ticket"
        className={`${baseClass}__screenshot`}
      />
    );

  return (
    <Modal
      title="Example ticket"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <div className={`${baseClass}`}>
        <p className="automations-learn-more">
          Want to learn more about how automations in Fleet work?{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/automations"
            text=" Check out the Fleet documentation"
            newTab
          />
        </p>
        <div className={`${baseClass}__example`}>{screenshot}</div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PreviewTicketModal;
