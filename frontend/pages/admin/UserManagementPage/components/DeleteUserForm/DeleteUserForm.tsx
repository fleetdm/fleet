import React, { useEffect } from "react";
import Button from "components/buttons/Button";

const baseClass = "delete-user-form";

interface IDeleteUserForm {
  name: string;
  onDelete: () => void;
  onCancel: () => void;
}

const DeleteUserForm = ({
  name,
  onDelete,
  onCancel,
}: IDeleteUserForm): JSX.Element => {
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onDelete();
      }
    };

    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  }, []);

  return (
    <div className={baseClass}>
      <p>
        You are about to delete{" "}
        <span className={`${baseClass}__name`}>{name}</span> from Fleet.
      </p>
      <p className={`${baseClass}__warning`}>This action cannot be undone.</p>
      <div className={`${baseClass}__btn-wrap`}>
        <Button
          className={`${baseClass}__btn`}
          type="button"
          variant="alert"
          onClick={onDelete}
        >
          Delete
        </Button>
        <Button
          className={`${baseClass}__btn`}
          onClick={onCancel}
          variant="inverse-alert"
        >
          Cancel
        </Button>
      </div>
    </div>
  );
};

export default DeleteUserForm;
