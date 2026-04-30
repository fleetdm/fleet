import React from "react";
import { ICommandResult } from "interfaces/command";
import CommandResultsModal, {
  ICommandResultsModalCommand,
  GetIconName,
} from "pages/hosts/components/CommandDetailsModal";
import { formatDistanceToNow } from "date-fns";
import IconStatusMessage from "components/IconStatusMessage";
import CustomLink from "components/CustomLink";

export interface IFailedEnrollmentProfileModalProps {
  command: Pick<ICommandResultsModalCommand, "command_uuid">;
  onDone: () => void;
}

const failedEnrollmentProfileContentBody = (
  baseClass: string,
  result: ICommandResult
) => {
  const displayTime = result.updated_at
    ? ` (${formatDistanceToNow(new Date(result.updated_at), {
        includeSeconds: true,
        addSuffix: true,
      })})`
    : null;
  const messageText = (
    <span>
      Fleet enrollment profile renewal failed for <b>{result.name}</b>
      {displayTime}.
    </span>
  );
  return (
    <div>
      <IconStatusMessage
        className={`${baseClass}__status-message`}
        iconName={GetIconName(result.status)}
        message={messageText}
      />
      <p>
        This profile contains a certificate that will expire. If the profile
        isn&apos;t renewed before expiration, the host must be re-enrolled. For
        assistance, reach out to{" "}
        <CustomLink
          text="Fleet support"
          url="https://fleetdm.com/support"
          newTab
        />
        .
      </p>
    </div>
  );
};

const FailedEnrollmentProfileModal = ({
  command,
  onDone,
}: IFailedEnrollmentProfileModalProps) => {
  return (
    <CommandResultsModal
      command={command}
      onDone={onDone}
      title="Enrollment profile renewal details"
      contentBody={failedEnrollmentProfileContentBody}
    />
  );
};

export default FailedEnrollmentProfileModal;
