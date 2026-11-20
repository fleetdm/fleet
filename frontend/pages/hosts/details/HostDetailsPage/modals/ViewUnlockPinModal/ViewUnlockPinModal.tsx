import { AxiosError } from "axios";
import React from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import { HumanTimeDiffWithDateTip } from "components/HumanTimeDiffWithDateTip";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import hostAPI, { IUnlockHostResponse } from "services/entities/hosts";

const baseClass = "view-unlock-pin-modal";

interface IViewUnlockPinModalProps {
  id: number;
  onClose: () => void;
}

const ViewUnlockPinModal = ({ id, onClose }: IViewUnlockPinModalProps) => {
  // Every open makes a new request, which logs a new viewed_host_unlock_pin
  // activity, so the response must not be cached.
  const { data, isLoading, isError } = useQuery<
    IUnlockHostResponse,
    AxiosError
  >(["android-unlock-pin", id], () => hostAPI.unlockHost(id), {
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: false,
    cacheTime: 0,
  });

  const renderContent = () => {
    if (isLoading) return <Spinner />;
    if (isError) return <DataError />;
    if (!data?.unlock_pin) return null;

    return (
      <>
        {data.locked_at && (
          <p>
            Fleet locked this host{" "}
            <HumanTimeDiffWithDateTip timeString={data.locked_at} />.
          </p>
        )}
        <div className={`${baseClass}__pin`}>
          <span>PIN:</span>
          <span className={`${baseClass}__pin-value`}>{data.unlock_pin}</span>
        </div>
        <p>
          If the PIN doesn&apos;t work, it was changed. To get back in, select{" "}
          <b>Actions &gt; Clear passcode</b>.
        </p>
      </>
    );
  };

  return (
    <Modal className={baseClass} title="View unlock PIN" onExit={onClose}>
      <div className={`${baseClass}__modal-content`}>{renderContent()}</div>
      <div className="modal-cta-wrap">
        <Button type="button" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
};

export default ViewUnlockPinModal;
