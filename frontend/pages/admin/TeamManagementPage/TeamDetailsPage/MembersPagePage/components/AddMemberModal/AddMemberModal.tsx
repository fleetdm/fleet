import React, { useCallback } from "react";

import Modal from "components/modals/Modal";
import Button from "../../../../../../../components/buttons/Button";
import AutocompleteDropdown from "../../../../../../../components/forms/fields/AutocompleteDropdown";
import { IEditTeamFormData } from "../../../../components/EditTeamModal/EditTeamModal";

const baseClass = "add-member-modal";

interface IAddMemberModal {
  onCancel: () => void;
  onSubmit: () => void;
}

const AddMemberModal = (props: IAddMemberModal) => {
  const { onCancel, onSubmit } = props;

  const onFormSubmit = useCallback(() => {
    onSubmit();
  }, [onSubmit]);

  return (
    <Modal onExit={onCancel} title={"Add Members"} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <AutocompleteDropdown
          id={"member-autocomplete"}
          value={[]}
          options={[{ value: "test", label: "test", disabled: false }]}
          onChange={(values) => console.log(values)}
          isLoading={false}
          placeholder={"Search users by name"}
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            Add Member
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default AddMemberModal;
