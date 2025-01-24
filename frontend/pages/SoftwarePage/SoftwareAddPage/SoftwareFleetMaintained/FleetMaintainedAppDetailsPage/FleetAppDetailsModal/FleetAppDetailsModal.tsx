import React from "react";
import { noop } from "lodash";

import Modal from "components/Modal";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";

const baseClass = "fleet-app-details-modal";

interface IFleetAppDetailsModalProps {
  name: string;
  platform: string;
  version: string;
  url?: string;
  onCancel: () => void;
}

const TOOLTIP_MESSAGE =
  "Fleet downloads the package from the URL and stores it. Hosts download it from Fleet before install.";

const FleetAppDetailsModal = ({
  name,
  platform,
  version,
  url,
  onCancel,
}: IFleetAppDetailsModalProps) => {
  return (
    <Modal className={baseClass} title="Software details" onExit={onCancel}>
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
              value={<TooltipTruncatedText value={url} />}
            />
          )}
        </div>
      </>
    </Modal>
  );
};

export default FleetAppDetailsModal;
