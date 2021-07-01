import React, { useState, useCallback } from "react";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IQuery } from "interfaces/query";

const baseClass = "schedule-editor-modal";

export interface IScheduleEditorFormData {
  name: string;
}

interface IDeleteScheduledQueryModalProps {
  queries: IQuery[];
  onCancel: () => void;
  onSubmit: (formData: IScheduleEditorFormData) => void;
}

const DeleteScheduledQueryModal = (
  props: IDeleteScheduledQueryModalProps
): JSX.Element => {
  const { onCancel, onSubmit, queries } = props;

  // FUNCTIONALITY LATER 7/1
  // const onFormSubmit = useCallback(
  //   (evt) => {
  //     evt.preventDefault();
  //     onSubmit({
  //       name,
  //     });
  //   },
  //   [onSubmit, name]
  // );

  return (
    <Modal title={"Schedule editor"} onExit={onCancel} className={baseClass}>
      <div>
        Are you sure you want to delete the selected queries?
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            // onClick={onFormSubmit}
          >
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default DeleteScheduledQueryModal;
