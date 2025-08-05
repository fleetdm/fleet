import React, { useEffect, useRef, useState } from "react";

import { IHostSoftware } from "interfaces/software";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";

import InventoryVersions from "../../components/InventoryVersions";

const baseClass = "inventory-versions-modal";

interface IInventoryVersionsModal {
  hostSoftware: IHostSoftware;
  onExit: () => void;
}

const InventoryVersionsModal = ({
  hostSoftware,
  onExit,
}: IInventoryVersionsModal) => {
  // For scrollable modal
  const [isTopScrolling, setIsTopScrolling] = useState(false);
  const topDivRef = useRef<HTMLDivElement>(null);
  const checkScroll = () => {
    if (topDivRef.current) {
      const isScrolling =
        topDivRef.current.scrollHeight > topDivRef.current.clientHeight;
      setIsTopScrolling(isScrolling);
    }
  };

  useEffect(() => {
    checkScroll();
    window.addEventListener("resize", checkScroll);
    return () => window.removeEventListener("resize", checkScroll);
  }, []);
  // end scrollable modal logic

  const renderScrollableContent = () => {
    return (
      <div className={`${baseClass}__content`} ref={topDivRef}>
        <InventoryVersions hostSoftware={hostSoftware} />
      </div>
    );
  };
  const renderFooter = () => (
    <ModalFooter
      isTopScrolling={isTopScrolling}
      primaryButtons={
        <Button type="submit" onClick={onExit}>
          Done
        </Button>
      }
    />
  );

  return (
    <Modal
      title={hostSoftware.name}
      className={baseClass}
      onExit={onExit}
      width="large"
    >
      <>
        {renderScrollableContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default InventoryVersionsModal;
