import React, { useCallback, useState } from "react";

import { IUser } from "interfaces/user";
import { INewMembersBody } from "interfaces/team";
import endpoints from "kolide/endpoints";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import AutocompleteDropdown from "components/forms/fields/AutocompleteDropdown";

const baseClass = "add-member-modal";

interface IAddMemberModal {
  onCancel: () => void;
  onSubmit: (userIds: INewMembersBody) => void;
}

const AddMemberModal = (props: IAddMemberModal): JSX.Element => {
  const { onCancel, onSubmit } = props;

  const [selectedMembers, setSelectedMembers] = useState([]);

  const onChangeDropdown = useCallback((values) => {
    setSelectedMembers(values);
  }, []);

  const onFormSubmit = useCallback(() => {
    const userIds = selectedMembers.map((member: IUser) => {
      return { id: member.id, role: "observer" };
    });
    onSubmit({ users: userIds });
  }, [selectedMembers, onSubmit]);

  return (
    <Modal onExit={onCancel} title={"Add Members"} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <AutocompleteDropdown
          id={"member-autocomplete"}
          resourceUrl={endpoints.USERS}
          onChange={onChangeDropdown}
          placeholder={"Search users by name"}
          value={selectedMembers}
          valueKey={"id"}
          labelKey={"name"}
        />
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            disabled={selectedMembers.length === 0}
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
