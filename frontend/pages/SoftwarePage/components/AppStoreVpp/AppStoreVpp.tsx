import React, { useState } from "react";
import { InjectedRouter } from "react-router";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";

const baseClass = "app-store-vpp";

interface IVppSoftwareListItemProps {
  software: any;
}

const VppSoftwareListItem = ({}: IVppSoftwareListItemProps) => {
  return <p>test</p>;
};

interface IVppSoftwareListProps {
  software: any[];
}

const VppSoftwareList = ({ software }: IVppSoftwareListProps) => {
  const renderContent = () => {
    if (software.length === 0) {
      return (
        <div className={`${baseClass}__no-software`}>
          <p className={`${baseClass}__no-software-title`}>
            You don&apos;t have any App Store apps
          </p>
          <p className={`${baseClass}__no-software-description`}>
            You must purchase apps in ABM. App Store apps that are already added
            to this team are not listed.
          </p>
        </div>
      );
    }

    return (
      <div className={`${baseClass}__software-list-items`}>
        {software.map((softwareItem) => (
          <p>test</p>
          // <VppSoftwareListItem key={software.id} software={software} />
        ))}
      </div>
    );
  };

  return <div className={`${baseClass}__software-list`}>{renderContent()}</div>;
};

interface IAppStoreVppProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
}

const AppStoreVpp = ({ teamId, router, onExit }: IAppStoreVppProps) => {
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);

  const renderContent = () => {
    const isLoading = false;
    if (isLoading) {
      return <Spinner />;
    }

    return <VppSoftwareList software={[]} />;
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Apple App Store apps purchased via Apple Business Manager.
      </p>
      {renderContent()}
      <div className="modal-cta-wrap">
        <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
          Add software
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );
};

export default AppStoreVpp;
