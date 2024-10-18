import React, { useMemo } from "react";

import { IMdmSolution } from "interfaces/mdm";

import Modal from "components/Modal";
import TableContainer from "components/TableContainer";
import Button from "components/buttons/Button";

import {
  generateSolutionsDataSet,
  generateSolutionsTableHeaders,
} from "./MdmSolutionModalTableConfig";

const baseClass = "mdm-solution-modal";

const SOLUTIONS_DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_TITLE = "Unknown MDM solution";

interface IMdmSolutionModalProps {
  mdmSolutions: IMdmSolution[];
  selectedPlatformLabelId?: number;
  selectedTeamId?: number;
  onCancel: () => void;
}

const MdmSolutionModal = ({
  mdmSolutions,
  selectedPlatformLabelId,
  selectedTeamId,
  onCancel,
}: IMdmSolutionModalProps) => {
  const solutionsTableHeaders = useMemo(
    () => generateSolutionsTableHeaders(selectedTeamId),
    [selectedTeamId]
  );
  const solutionsDataSet = generateSolutionsDataSet(
    mdmSolutions,
    selectedPlatformLabelId
  );

  return (
    <Modal
      className={baseClass}
      title={mdmSolutions[0].name || DEFAULT_TITLE}
      width="large"
      onExit={onCancel}
      onEnter={onCancel}
    >
      <>
        <div className={`${baseClass}__modal-content`}>
          <TableContainer
            isLoading={false}
            emptyComponent={() => null} // if this modal is shown, this table should never be empty
            columnConfigs={solutionsTableHeaders}
            data={solutionsDataSet}
            defaultSortHeader={SOLUTIONS_DEFAULT_SORT_HEADER}
            defaultSortDirection={DEFAULT_SORT_DIRECTION}
            resultsTitle="MDM"
            showMarkAllPages={false}
            isAllPagesSelected={false}
            disableCount
            disablePagination
            disableTableHeader
          />
        </div>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default MdmSolutionModal;
