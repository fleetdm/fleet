import React from "react";
import { noop } from "lodash";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "fleet-app-details-modal";

interface IFleetAppDetailsModalProps {
  name: string;
  platform: string;
  version: string;
  url?: string;
}

const TOOLTIP_MESSAGE =
  "Fleet downloads the package from the URL and stores it. Hosts download it from Fleet before install.";

const FleetAppDetailsModal = ({
  name,
  platform,
  version,
  url,
}: IFleetAppDetailsModalProps) => {
  return (
    <Modal className={baseClass} title="Software details" onExit={noop}>
      <>
        <div className={`${baseClass}__modal-content`}>
          <DataSet title="Name" value={name} />
          <DataSet title="Platform" value={platform} />
          <DataSet title="Version" value={version} />
          {url && (
            <DataSet
              title={
                <TooltipWrapper tipContent={TOOLTIP_MESSAGE}>
                  URL
                </TooltipWrapper>
              }
              value={url}
            />
          )}
        </div>
      </>
    </Modal>
  );
};

export default FleetAppDetailsModal;
