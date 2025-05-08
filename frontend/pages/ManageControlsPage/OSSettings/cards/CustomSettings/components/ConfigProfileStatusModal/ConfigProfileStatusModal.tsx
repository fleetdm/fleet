import React from "react";
import { useQuery } from "react-query";

import configProfileAPI from "services/entities/config_profiles";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Button from "components/buttons/Button";
import ConfigProfileStatusTable from "../ConfigProfileStatusTable";

const baseClass = "config-profile-status-modal";

interface IConfigProfileStatusModalProps {
  name: string;
  uuid: string;
  teamId: number;
  onExit: () => void;
}

const ConfigProfileStatusModal = ({
  name,
  uuid,
  teamId,
  onExit,
}: IConfigProfileStatusModalProps) => {
  const { data, isLoading, isError } = useQuery(
    ["config-profile-status", uuid],
    () => configProfileAPI.getConfigProfileStatus(uuid),
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

    return (
      <ConfigProfileStatusTable
        teamId={teamId}
        uuid={uuid}
        profileStatus={data}
      />
    );
  };

  return (
    <Modal className={baseClass} title={name} onExit={onExit}>
      <>
        {renderContent()}
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ConfigProfileStatusModal;
