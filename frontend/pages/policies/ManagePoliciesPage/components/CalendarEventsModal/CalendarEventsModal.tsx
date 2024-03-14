import React, { useCallback, useState } from "react";

import { IPolicy } from "interfaces/policy";

import validURL from "components/forms/validators/valid_url";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Graphic from "components/Graphic";
import Modal from "components/Modal";

const baseClass = "calendar-events-modal";

interface ICalendarEventsModal {
  onExit: () => void;
  onSubmit: () => void;
  configured: boolean;
  enabled: boolean;
  url: string;
  policies: IPolicy[];
  enabledPolicies: number[];
}

interface IFormPolicy {
  name: string;
  id: number;
  checked: boolean;
}
interface ICalendarEventsFormData {
  enabled: boolean;
  url: string;
  policies: IFormPolicy[];
}

// allows any policy name to be the name of a form field, one of the checkboxes
type FormNames = string;

const CalendarEventsModal = ({
  onExit,
  onSubmit,
  configured,
  enabled,
  url,
  policies,
  enabledPolicies,
}: ICalendarEventsModal) => {
  const [formData, setFormData] = useState<ICalendarEventsFormData>({
    enabled,
    url,
    // TODO - stay udpdated on state of backend approach to syncing policies in the policies table
    // and in the new calendar table
    // id may change if policy was deleted
    // name could change if policy was renamed
    policies: policies.map((policy) => ({
      name: policy.name,
      id: policy.id,
      checked: enabledPolicies.includes(policy.id),
    })),
  });
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showPreviewCalendarEvent, setShowPreviewCalendarEvent] = useState(
    false
  );

  const validateCalendarEventsFormData = (
    curFormData: ICalendarEventsFormData
  ) => {
    const errors: Record<string, string> = {};
    const { url: curUrl } = curFormData;
    if (!validURL({ url: curUrl })) {
      const errorPrefix = curUrl ? `${curUrl} is not` : "Please enter";
      errors.resolutionWebhookUrl = `${errorPrefix} a valid resolution webhook URL`;
    }
    return {};
  };

  const onInputChange = useCallback(
    (newVal: { name: FormNames; value: string | number | boolean }) => {
      const { name, value } = newVal;
      let newFormData: ICalendarEventsFormData;
      if (["enabled", "url"].includes(name)) {
        newFormData = { ...formData, [name]: value };
      } else if (typeof value === "boolean") {
        const newFormPolicies = formData.policies.map((formPolicy) => {
          if (formPolicy.name === name) {
            return { ...formPolicy, checked: value };
          }
          return formPolicy;
        });
        newFormData = { ...formData, policies: newFormPolicies };
      } else {
        throw TypeError("Unexpected value type for policy checkbox");
      }
      setFormData(newFormData);
      setFormErrors(validateCalendarEventsFormData(newFormData));
    },
    [formData]
  );

  const togglePreviewCalendarEvent = () => {
    // TODO
  };

  const renderPoliciesList = () => {
    //  TODO
  };
  const renderPreviewCalendarEventModal = () => {
    // TODO
    return <></>;
  };

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

  const renderConfiguredModal = () => (
    <div className={`${baseClass} form`}>
      <Slider
        value={formData.enabled}
        onChange={() => {
          onInputChange({ name: "enabled", value: !formData.enabled });
        }}
        inactiveText="Disabled"
        activeText="Enabled"
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
        value={formData.url}
        parseTarget
        error={formErrors.url}
        tooltip="Provide a URL to deliver a webhook request to."
        helpText="A request will be sent to this URL during the calendar event. Use it to trigger auto-remidiation."
      />
      {/* <RevealButton
        isShowing={showExamplePayload}
        className={`${baseClass}__show-example-payload-toggle`}
        hideText="Hide example payload"
        showText="Show example payload"
        caretPosition="after"
        onClick={() => {
          setShowExamplePayload(!showExamplePayload);
        }}
      />
      {showExamplePayload && renderExamplePayload()} */}
      {renderPoliciesList()}
    </div>
  );

  if (showPreviewCalendarEvent) {
    return renderPreviewCalendarEventModal();
  }
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
