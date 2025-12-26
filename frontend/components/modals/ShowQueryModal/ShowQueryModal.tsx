import React from "react";

import SQLEditor from "components/SQLEditor";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import { PerformanceImpactIndicator } from "interfaces/performance_impact_indicator";

const baseClass = "show-query-modal";

interface IShowQueryModalProps {
  onCancel: () => void;
  query?: string;
  impact?: PerformanceImpactIndicator;
}

const ShowQueryModal = ({
  query,
  impact,
  onCancel,
}: IShowQueryModalProps): JSX.Element => {
  return (
    <Modal
      title="Query"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={baseClass}>
        <SQLEditor
          value={query}
          name="Query"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          wrapEnabled
          readOnly
        />
        {impact && (
          <div className={`${baseClass}__performance-impact`}>
            Performance impact:{" "}
            <PerformanceImpactCell value={{ indicator: impact }} />
          </div>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default ShowQueryModal;
