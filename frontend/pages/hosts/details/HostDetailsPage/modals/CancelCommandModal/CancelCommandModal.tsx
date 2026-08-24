import React from "react";
import { noop } from "lodash";

import { ICommand } from "interfaces/command";
import commandsAPI from "services/entities/command";

import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

import CommandItem from "pages/hosts/details/cards/Activity/CommandItem/CommandItem";

const baseClass = "cancel-command-modal";

interface ICancelCommandModalProps {
  hostId: number;
  command: ICommand;
  onSuccessCancel: (command: ICommand) => void;
  onExit: () => void;
}

const CancelCommandModal = ({
  hostId,
  command,
  onSuccessCancel,
  onExit,
}: ICancelCommandModalProps) => {
  const [isCanceling, setIsCanceling] = React.useState(false);

  const onAttemptCancel = async () => {
    setIsCanceling(true);
    try {
      await commandsAPI.cancelHostCommand(hostId, command.command_uuid);
      notify.success(
        `Successfully canceled the ${command.request_type} command.`
      );
      onSuccessCancel(command);
    } catch (err) {
      notify.error(
        `Couldn't cancel the ${command.request_type} command. Please try again.`,
        { response: err }
      );
    }
    onExit();
  };

  return (
    <Modal
      className={baseClass}
      title="Cancel upcoming command"
      onExit={onExit}
      isContentDisabled={isCanceling}
    >
      <div className={`${baseClass}__content`}>
        <p>
          If the activity is happening on the host it will still complete.
          Results won&apos;t appear in Fleet.
        </p>
        <CommandItem command={command} onShowDetails={noop} isSoloItem />
      </div>
      <div className="modal-cta-wrap">
        <Button
          disabled={isCanceling}
          isLoading={isCanceling}
          variant="alert"
          onClick={onAttemptCancel}
        >
          Cancel command
        </Button>
        <Button onClick={onExit} variant="secondary">
          Back
        </Button>
      </div>
    </Modal>
  );
};

export default CancelCommandModal;
