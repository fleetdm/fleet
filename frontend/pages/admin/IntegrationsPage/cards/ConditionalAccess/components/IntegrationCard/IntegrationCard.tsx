import React from "react";
import classnames from "classnames";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "integration-card";

export interface IIntegrationCardProps {
  provider: "okta" | "microsoft-entra";
  title: string;
  description: string;
  isConfigured: boolean;
  isPending?: boolean; // Awaiting external action (e.g., OAuth completion)
  isLoading?: boolean;
  configuredInfo?: string; // e.g., tenant ID to display when configured
  onConnect: () => void;
  onEdit?: () => void;
  onDelete?: () => void;
  className?: string;
}

const IntegrationCard = ({
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  provider: _,
  title,
  description,
  isConfigured,
  isPending = false,
  isLoading = false,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  configuredInfo: __,
  onConnect,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  onEdit: ___,
  onDelete,
  className,
}: IIntegrationCardProps) => {
  const cardClasses = classnames(baseClass, className);

  const renderUnconfiguredState = () => (
    <>
      <div className={`${baseClass}__content-wrapper`}>
        <div className={`${baseClass}__content`}>
          <h3 className={`${baseClass}__title`}>{title}</h3>
          <p className={`${baseClass}__description`}>{description}</p>
        </div>
      </div>
      <div className={`${baseClass}__cta`}>
        <Button onClick={onConnect} isLoading={isLoading}>
          Connect
        </Button>
      </div>
    </>
  );

  const renderPendingState = () => (
    <>
      <div className={`${baseClass}__content-wrapper`}>
        <Icon name="pending-outline" />
        <div className={`${baseClass}__content`}>
          <h3 className={`${baseClass}__title`}>{title}</h3>
          <p className={`${baseClass}__description`}>{description}</p>
        </div>
      </div>
    </>
  );

  const renderConfiguredState = () => (
    <>
      <div className={`${baseClass}__content-wrapper`}>
        <Icon name="success" />
        <div className={`${baseClass}__content`}>
          <p className={`${baseClass}__configured-text`}>
            {title} conditional access configured
          </p>
        </div>
      </div>
      <div className={`${baseClass}__cta`}>
        {onDelete && (
          <Button variant="text-icon" onClick={onDelete}>
            Delete
            <Icon name="trash" color="ui-fleet-black-75" />
          </Button>
        )}
      </div>
    </>
  );

  const renderContent = () => {
    if (isConfigured) {
      return renderConfiguredState();
    }
    if (isPending) {
      return renderPendingState();
    }
    return renderUnconfiguredState();
  };

  return (
    <Card className={cardClasses} color="grey">
      {renderContent()}
    </Card>
  );
};

export default IntegrationCard;
