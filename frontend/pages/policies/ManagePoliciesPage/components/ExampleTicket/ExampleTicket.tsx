import React, { useContext } from "react";

import { AppContext } from "context/app";

import { IIntegrationType } from "interfaces/integration";

import Card from "components/Card";
import JiraPreview from "../../../../../../assets/images/jira-policy-automation-preview-400x419@2x.png";
import ZendeskPreview from "../../../../../../assets/images/zendesk-policy-automation-preview-400x515@2x.png";
import JiraPreviewPremium from "../../../../../../assets/images/jira-policy-automation-preview-premium-400x316@2x.png";
import ZendeskPreviewPremium from "../../../../../../assets/images/zendesk-policy-automation-preview-premium-400x483@2x.png";

const baseClass = "example-ticket";

interface IExampleTicketProps {
  integrationType?: IIntegrationType;
}

const ExampleTicket = ({
  integrationType,
}: IExampleTicketProps): JSX.Element => {
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
    <Card className={baseClass} color="gray">
      {screenshot}
    </Card>
  );
};

export default ExampleTicket;
