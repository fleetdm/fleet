import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import YamlAce from "components/YamlAce";

const baseClass = "show-query-modal";

interface IShowQueryModalProps {
  onCancel: () => void;
  liveQuery: string;
}

const ShowQueryModal = ({
  onCancel,
  liveQuery,
}: IShowQueryModalProps): JSX.Element => {
  const handleAceInputChange = (value: string) => {};

  return (
    <Modal title={"Query"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__show-query-modal`}>
        <YamlAce
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          onChange={handleAceInputChange}
          name="liveQuery"
          value={liveQuery}
        />

        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="brand"
          >
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ShowQueryModal;
