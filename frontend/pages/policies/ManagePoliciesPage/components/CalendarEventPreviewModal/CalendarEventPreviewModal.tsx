/* This component is used for creating and editing both global and team scheduled queries */

import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { IPolicy } from "interfaces/policy";

const baseClass = "calendar-event-preview-modal";

interface ICalendarEventPreviewModalProps {
  onCancel: () => void;
  policy?: IPolicy;
}

const CalendarEventPreviewModal = ({
  onCancel,
  policy,
}: ICalendarEventPreviewModalProps): JSX.Element => {
  return (
    <Modal
      title="Calendar event preview"
      width="large"
      onExit={onCancel}
      className={baseClass}
    >
      <>
        <p>
          {policy ? (
            <>
              End users failing only <strong>{policy.name}</strong> policy will
              see:
            </>
          ) : (
            "What end users see:"
          )}
        </p>
        <div className={`${baseClass}__preview`}>
          <div className={`${baseClass}__preview-header`}>
            <div className={`${baseClass}__preview-header__square`}>
              <div />
            </div>
            <div className={`${baseClass}__preview-header__info`}>
              Scheduled maintenance
            </div>
          </div>
          <div className={`${baseClass}__preview-info`}>
            <div className={`${baseClass}__preview-info__icon`}>
              <div />
            </div>
            <div className={`${baseClass}__preview-info__text`}>
              <p>
                Acme, Inc. reserved this time to make some changes to
                Anna&apos;s MacBook Pro. Please leave your computer on and
                connected to power.
              </p>
              <p>
                <strong>Why it matters</strong>
                <br />
                {policy?.description}
              </p>
              <p>
                <strong>What we&apos;ll do</strong>
                <br />
                {policy?.resolution}
              </p>
            </div>
          </div>
          <div className={`${baseClass}__preview-invitee`}>
            <div className={`${baseClass}__preview-invitee__icon`}>
              <Icon name="calendar" />
            </div>
            <div className={`${baseClass}__preview-invitee__text`}>
              Anna Chao
            </div>
          </div>
        </div>
        {policy ? (
          <p>
            <strong>Why it matters</strong> and{" "}
            <strong>What we&apos;ll do</strong> are populated by the
            policy&apos;s <strong>Description</strong> and{" "}
            <strong>Resolution</strong> respectively.
          </p>
        ) : (
          <p>
            Users failing only a single policy will see a more specific
            explanation.
          </p>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default CalendarEventPreviewModal;
