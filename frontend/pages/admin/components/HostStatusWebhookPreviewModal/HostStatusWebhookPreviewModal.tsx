import React from "react";

import { syntaxHighlight } from "utilities/helpers";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "host-status-webhook-preview-modal";

const getHostStatusPreview = (teamScope?: boolean) => {
  const data = {
    unseen_hosts: 1,
    total_hosts: 2,
    days_unseen: 3,
    team_id: 123,
  } as Record<string, number>;

  if (!teamScope) {
    delete data.team_id;
  }

  return {
    text:
      "More than X% of your hosts have not checked into Fleet for more than Y days. You’ve been sent this message because the Host status webhook is enabled in your Fleet instance.",
    data,
  };
};

interface IHostStatusWebhookPreviewModal {
  isTeamScope?: boolean;
  toggleModal: () => void;
}

const HostStatusWebhookPreviewModal = ({
  isTeamScope = false,
  toggleModal,
}: IHostStatusWebhookPreviewModal) => {
  return (
    <Modal
      title="Host status webhook"
      onExit={toggleModal}
      onEnter={toggleModal}
      className={baseClass}
    >
      <>
        <p>
          An example request sent to your configured <b>Destination URL</b>.
        </p>
        <div className={baseClass}>
          <pre
            dangerouslySetInnerHTML={{
              __html: syntaxHighlight(getHostStatusPreview(isTeamScope)),
            }}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={toggleModal} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default HostStatusWebhookPreviewModal;
