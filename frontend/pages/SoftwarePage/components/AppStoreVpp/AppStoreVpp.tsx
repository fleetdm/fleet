import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Radio from "components/forms/fields/Radio";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "app-store-vpp";

interface IVppSoftwareListItemProps {
  software: any;
  selected: boolean;
  onSelect: (software: any) => void;
}

const VppSoftwareListItem = ({
  software,
  selected,
  onSelect,
}: IVppSoftwareListItemProps) => {
  return (
    <li className={`${baseClass}__list-item`}>
      <Radio
        label="Test software"
        id="test-software"
        checked={selected}
        value={"test-software "}
        name="vppSoftware"
        onChange={onSelect}
      />
    </li>
  );
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
      <ul className={`${baseClass}__list-items`}>
        {software.map((softwareItem) => (
          <VppSoftwareListItem
            key={softwareItem.id}
            software={softwareItem}
            selected={false}
            onSelect={() => {}}
          />
        ))}
      </ul>
    );
  };

  return <div className={`${baseClass}__list`}>{renderContent()}</div>;
};

interface IAppStoreVppProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
}

const AppStoreVpp = ({ teamId, router, onExit }: IAppStoreVppProps) => {
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);

  const { data, isLoading, isError } = useQuery("vppSoftware", () => {}, {
    ...DEFAULT_USE_QUERY_OPTIONS,
  });

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError className={`${baseClass}__error`} />;
    }

    return <VppSoftwareList software={[1, 2, 3]} />;
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
