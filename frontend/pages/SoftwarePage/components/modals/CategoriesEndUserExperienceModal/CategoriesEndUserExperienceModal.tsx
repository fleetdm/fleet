import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CategoriesEndUserExperiencePreview from "../../../../../../assets/images/categories-end-user-experience-preview-570x259@2x.png";
import CategoriesEndUserExperiencePreviewMobile from "../../../../../../assets/images/categories-end-user-experience-preview-mobile@2x.png";

const baseClass = "categories-end-user-experience-preview-modal";

interface ICategoriesEndUserExperienceModal {
  onCancel: () => void;
  isIosOrIpadosApp?: boolean;
}

const CategoriesEndUserExperienceModal = ({
  onCancel,
  isIosOrIpadosApp = false,
}: ICategoriesEndUserExperienceModal): JSX.Element => {
  console.log("isIosOrIpadosApp", isIosOrIpadosApp);
  return (
    <Modal title="End user experience" onExit={onCancel} className={baseClass}>
      <>
        <span>What end users see:</span>
        <div className={`${baseClass}__preview`}>
          <img
            src={
              isIosOrIpadosApp
                ? CategoriesEndUserExperiencePreviewMobile
                : CategoriesEndUserExperiencePreview
            }
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
