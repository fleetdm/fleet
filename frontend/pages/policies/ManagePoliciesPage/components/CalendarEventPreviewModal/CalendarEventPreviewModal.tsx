import React, { useContext } from "react";

import { TRANSPARENCY_LINK } from "utilities/constants";

import { AppContext } from "context/app";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { IFormPolicy } from "../PoliciesPaginatedList/PoliciesPaginatedList";

const baseClass = "calendar-event-preview-modal";

interface ICalendarEventPreviewModalProps {
  onCancel: () => void;
  policy?: IFormPolicy;
}

const CalendarEventPreviewModal = ({
  onCancel,
  policy,
}: ICalendarEventPreviewModalProps): JSX.Element => {
  const { config } = useContext(AppContext);

  const showGenericPreview = !policy?.description || !policy?.resolution;
  const orgName = config?.org_info.org_name;

  return (
    <Modal
      title="Calendar event preview"
      width="large"
      onExit={onCancel}
      className={baseClass}
    >
      <>
        <span>
          {showGenericPreview ? (
            "What end users see:"
          ) : (
            <>
              End users failing only <strong>{policy.name}</strong> policy will
              see:
            </>
          )}
        </span>
        <div className={`${baseClass}__preview`}>
          <div className={`${baseClass}__preview-header`}>
            <div className={`${baseClass}__preview-header__square-wrapper`}>
              <div className={`${baseClass}__preview-header__square`} />
            </div>
            <div className={`${baseClass}__preview-header__info`}>
              <div className={`${baseClass}__preview-header__title`}>
                💻 🚫 Scheduled maintenance
              </div>
              <div className={`${baseClass}__preview-header__time`}>
                <span>Tuesday, June 18</span>
                <span>⋅</span>
                <span>5-5:30pm</span>
              </div>
            </div>
          </div>
          <div className={`${baseClass}__preview-info`}>
            <div className={`${baseClass}__preview-info__icon`}>
              <Icon name="text" />
            </div>
            <div className={`${baseClass}__preview-info__text`}>
              {orgName} reserved this time to make some changes to your work
              computer (Anna&apos;s MacBook Pro).
              <br />
              <br />
              Please leave your device on and connected to power.
              <br /> <br />
              <strong>Why it matters</strong>
              <br />
              <div className={`${baseClass}__preview-info__text__user-text`}>
                {showGenericPreview
                  ? `${orgName} needs to make sure your device meets the organization's requirements.`
                  : policy.description}
              </div>
              <br /> <br />
              <strong>Maintenance required</strong>
              <br />
              <div className={`${baseClass}__preview-info__text__user-text`}>
                {showGenericPreview ? (
                  <ul>
                    <li>
                      Click the{" "}
                      <a
                        href={TRANSPARENCY_LINK}
                        rel="noreferrer"
                        target="_blank"
                      >
                        Fleet
                      </a>{" "}
                      icon in your computer&apos;s menu and select{" "}
                      <b>My device</b>
                    </li>
                    <li>
                      Navigate to the <b>Policies</b> tab
                    </li>
                    <li>
                      Follow instructions to resolve any policies marked{" "}
                      {`"No"`}
                    </li>
                    <li>
                      Click <b>Refetch</b>
                    </li>
                  </ul>
                ) : (
                  policy.resolution
                )}
              </div>
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
        <div className={`${baseClass}__footer`}>
          {showGenericPreview ? (
            <>
              Users failing only a single policy will see a more specific
              explanation.
            </>
          ) : (
            <>
              <strong>Why it matters</strong> and{" "}
              <strong>Maintenance required</strong> are populated by the
              policy&apos;s <strong>Description</strong> and{" "}
              <strong>Resolution</strong> respectively.
            </>
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default CalendarEventPreviewModal;
