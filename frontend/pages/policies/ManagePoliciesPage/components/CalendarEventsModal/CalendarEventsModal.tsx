import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Graphic from "components/Graphic";
import Modal from "components/Modal";
import React from "react";

const baseClass = "calendar-events-modal";

interface ICalendarEventsModal {
  onExit: () => void;
  onSubmit: () => void;
  configured: boolean;
}

const CalendarEventsModal = ({
  onExit,
  onSubmit,
  configured,
}: ICalendarEventsModal) => {
  const renderPlaceholderModal = () => {
    return (
      <>
        <a href="https://www.fleetdm.com/learn-more-about/calendar-events">
          <Graphic name="calendar-integration-not-configured" />
        </a>
        To create calendar events for end users if their hosts fail policies,
        you must first connect Fleet to your Google Workspace service account.
        <br />
        This can be configured in{" "}
        <b>Settings &gt; Integrations &gt; Calendars.</b>
        <CustomLink
          url="https://www.fleetdm.com/learn-more-about/calendar-events"
          text="Learn more"
          newTab
        />
        <div className="modal-cta-wrap">
          <Button onClick={onExit} variant="brand">
            Done
          </Button>
        </div>
      </>
    );
  };
  const renderConfiguredModal = () => {
    return <></>;
  };
  return (
    <Modal
      title="Calendar events"
      onExit={onExit}
      onEnter={configured ? onSubmit : onExit}
      className={baseClass}
    >
      {configured ? renderConfiguredModal() : renderPlaceholderModal()}
    </Modal>
  );
};

export default CalendarEventsModal;
