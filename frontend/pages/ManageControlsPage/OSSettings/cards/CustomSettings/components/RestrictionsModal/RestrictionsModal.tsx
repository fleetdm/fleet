import React from "react";
import { useQuery } from "react-query";

import configProfileAPI from "services/entities/config_profiles";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import ConfigProfileStatusTable from "../ConfigProfileStatusTable";

const baseClass = "restrictions-modal";

interface IRestrictionsModalProps {
  profileUUID: string;
  onExit: () => void;
}

const RestrictionsModal = ({
  profileUUID,
  onExit,
}: IRestrictionsModalProps) => {
  const { data, isLoading, isError } = useQuery(
    ["config-profile-status", profileUUID],
    () => configProfileAPI.getConfigProfileStatus(profileUUID),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (isError) {
      return <DataError verticalPaddingSize="pad-medium" />;
    }

    if (!data) {
      return null;
    }

    return <ConfigProfileStatusTable profileStatus={data} />;
  };

  return (
    <Modal className={baseClass} title="Restrictions" onExit={onExit}>
      <>
        {renderContent()}
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default RestrictionsModal;
