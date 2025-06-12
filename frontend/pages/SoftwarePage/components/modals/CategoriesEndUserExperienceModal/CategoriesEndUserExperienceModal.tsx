import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CategoriesEndnUserExperiencePreview from "../../../../../../assets/images/categories-end-user-experience-preview-570x259@2x.png";

const baseClass = "categories-end-user-experience-preview-modal";

interface ICategoriesEndUserExperienceModal {
  onCancel: () => void;
}

const CategoriesEndUserExperienceModal = ({
  onCancel,
}: ICategoriesEndUserExperienceModal): JSX.Element => {
  return (
    <Modal title="End user experience" onExit={onCancel} className={baseClass}>
      <>
        <span>What end users see:</span>
        <div className={`${baseClass}__preview`}>
          <img
            src={CategoriesEndnUserExperiencePreview}
            alt="Categories end user experience preview"
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CategoriesEndUserExperienceModal;
