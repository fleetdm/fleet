import React, { useEffect, useRef, useState } from "react";

import { IGeoLocation } from "interfaces/host";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";

const baseClass = "location-modal";

interface ILocationModal {
  hostGeolocation?: IGeoLocation;
  onExit: () => void;
}

const LocationModal = ({ hostGeolocation, onExit }: ILocationModal) => {
  if (!hostGeolocation) {
    return null;
  }

  const renderContent = () => {
    return (
      <div className={`${baseClass}__content`}>{hostGeolocation.city_name}</div>
    );
  };
  const renderFooter = () => (
    <ModalFooter
      primaryButtons={
        <Button type="submit" onClick={onExit}>
          Done
        </Button>
      }
    />
  );

  return (
    <Modal title="Location" className={baseClass} onExit={onExit} width="large">
      <>
        {renderContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default LocationModal;
