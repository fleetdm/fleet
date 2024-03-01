import React from "react";

import { IMdmSolution } from "interfaces/mdm";

import Modal from "components/Modal";

const baseClass = "mdm-solution-modal";

interface IMdmSolutionModalProps {
  mdmSolutions: IMdmSolution[];
  onCancel: () => void;
}

const MdmSolutionsModal = ({
  mdmSolutions,
  onCancel,
}: IMdmSolutionModalProps) => {
  console.log(mdmSolutions);

  return (
    <Modal className={baseClass} title="test" onExit={onCancel}>
      <div>test</div>
    </Modal>
  );
};

export default MdmSolutionsModal;
