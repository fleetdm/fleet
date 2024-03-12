import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import SelectTargetsDropdownStories from "components/forms/fields/SelectTargetsDropdown/SelectTargetsDropdown.stories";
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
    return;
    <div className={`${baseClass} form`}>
      <Slider
        value={calEventsEnabled}
        onChange={() => {
          setCalEventsEnabled(!calEventsEnabled);
          SelectTargetsDropdownStories({});
        }}
      />
      <Button
        type="button"
        variant="text-link"
        onClick={togglePreviewCalendarEvent}
      >
        Preview calendar event
      </Button>
      <InputField
        placeholder="https://server.com/example"
        label="Resolution webhook URL"
        onChange={onInputChange}
        name="resolutionWebhookUrl"
        value={formData.resolutionWebhookUrl}
        parseTarget
        error={formErrors.resolutionWebhookUrl}
        // TODO - update tooltip
        tooltip={<>TBD</>}
        helpText="A request will be sent to this URL during the calendar event. Use it to trigger auto-remidiation."
      />
      <RevealButton
        isShowing={showExamplePayload}
        className={`${baseClass}__show-example-payload-toggle`}
        hideText="Hide example payload"
        showText="Show example payload"
        caretPosition="after"
        onClick={() => {
          setShowExamplePayload(!showExamplePayload);
        }}
      />
      {showExamplePayload && renderExamplePayload()}
      {renderPoliciesList()}
    </div>;
  };
  return showPreviewCalendarEvent ? (
    renderPreviewCalendarEventModal()
  ) : (
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
