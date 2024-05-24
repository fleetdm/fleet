import React, { useCallback, useState } from "react";

import { IPolicy } from "interfaces/policy";

import validURL from "components/forms/validators/valid_url";

import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Checkbox from "components/forms/fields/Checkbox";
import { syntaxHighlight } from "utilities/helpers";
import Icon from "components/Icon";
import CalendarEventPreviewModal from "../CalendarEventPreviewModal";
import CalendarPreview from "../../../../../../assets/images/calendar-preview-720x436@2x.png";

const baseClass = "calendar-events-modal";

interface IFormPolicy {
  name: string;
  id: number;
  isChecked: boolean;
}
export interface ICalendarEventsFormData {
  enabled: boolean;
  url: string;
  policies: IFormPolicy[];
}

interface ICalendarEventsModal {
  onExit: () => void;
  updatePolicyEnabledCalendarEvents: (
    formData: ICalendarEventsFormData
  ) => void;
  isUpdating: boolean;
  configured: boolean;
  enabled: boolean;
  url: string;
  policies: IPolicy[];
}

// allows any policy name to be the name of a form field, one of the checkboxes
type FormNames = string;

const CalendarEventsModal = ({
  onExit,
  updatePolicyEnabledCalendarEvents,
  isUpdating,
  configured,
  enabled,
  url,
  policies,
}: ICalendarEventsModal) => {
  const [formData, setFormData] = useState<ICalendarEventsFormData>({
    enabled,
    url,
    policies: policies.map((policy) => ({
      name: policy.name,
      id: policy.id,
      isChecked: policy.calendar_events_enabled || false,
    })),
  });
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showPreviewCalendarEvent, setShowPreviewCalendarEvent] = useState(
    false
  );
  const [showExamplePayload, setShowExamplePayload] = useState(false);
  const [selectedPolicyToPreview, setSelectedPolicyToPreview] = useState<
    IPolicy | undefined
  >();

  // Used on URL change only when URL error exists and always on attempting to save
  const validateForm = (newFormData: ICalendarEventsFormData) => {
    const errors: Record<string, string> = {};
    const { url: newUrl } = newFormData;
    if (
      formData.enabled &&
      !validURL({ url: newUrl || "", protocols: ["http", "https"] })
    ) {
      const errorPrefix = newUrl ? `${newUrl} is not` : "Please enter";
      errors.url = `${errorPrefix} a valid resolution webhook URL`;
    }

    return errors;
  };

  const onFeatureEnabledChange = () => {
    const newFormData = { ...formData, enabled: !formData.enabled };

    const isDisabling = newFormData.enabled === false;

    // On disabling feature, validate URL and if an error clear input and error
    if (isDisabling) {
      const errors = validateForm(newFormData);

      if (errors.url) {
        newFormData.url = "";
        delete formErrors.url;
        setFormErrors(formErrors);
      }
    }

    setFormData(newFormData);
  };

  const onUrlChange = (value: string) => {
    const newFormData = { ...formData, url: value };
    // On URL change with erroneous URL, validate form
    if (formErrors.url) {
      setFormErrors(validateForm(newFormData));
    }

    setFormData(newFormData);
  };

  const onPolicyEnabledChange = useCallback(
    (newVal: { name: FormNames; value: boolean }) => {
      const { name, value } = newVal;
      const newFormPolicies = formData.policies.map((formPolicy) => {
        if (formPolicy.name === name) {
          return { ...formPolicy, isChecked: value };
        }
        return formPolicy;
      });
      const newFormData = { ...formData, policies: newFormPolicies };
      setFormData(newFormData);
    },
    [formData]
  );

  const onUpdatePolicyEnabledCalendarEvents = () => {
    const errors = validateForm(formData);

    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
    } else {
      updatePolicyEnabledCalendarEvents(formData);
    }
  };

  const togglePreviewCalendarEvent = () => {
    setShowPreviewCalendarEvent(!showPreviewCalendarEvent);
  };

  const renderExamplePayload = () => {
    return (
      <>
        <pre>POST https://server.com/example</pre>
        <pre
          dangerouslySetInnerHTML={{
            __html: syntaxHighlight({
              timestamp: "0000-00-00T00:00:00Z",
              host_id: 1,
              host_display_name: "Anna's MacBook Pro",
              host_serial_number: "ABCD1234567890",
              failing_policies: [
                {
                  id: 123,
                  name: "macOS - Disable guest account",
                },
              ],
            }),
          }}
        />
      </>
    );
  };

  const renderPolicies = () => {
    return (
      <div className="form-field">
        <div className="form-field__label">Policies:</div>
        <div className="automated-policies-section">
          {formData.policies.map((policy) => {
            const { isChecked, name, id } = policy;
            return (
              <div className="checkbox-row" id={`checkbox-row--${id}`} key={id}>
                <Checkbox
                  value={isChecked}
                  name={name}
                  // can't use parseTarget as value needs to be set to !currentValue
                  onChange={() => {
                    onPolicyEnabledChange({ name, value: !isChecked });
                  }}
                  smallTick
                >
                  {name}
                </Checkbox>
                <div>
                  <Button
                    variant="text-icon"
                    onClick={() => {
                      setSelectedPolicyToPreview(
                        policies.find((p) => p.id === id)
                      );
                      togglePreviewCalendarEvent();
                    }}
                    className="checkbox-row__preview-button"
                  >
                    <Icon name="eye" /> Preview
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
        <span className="form-field__help-text">
          A calendar event will be created for end users if one of their hosts
          fail any of these policies.{" "}
          <CustomLink
            url="https://www.fleetdm.com/learn-more-about/calendar-events"
            text="Learn more"
            newTab
          />
        </span>
      </div>
    );
  };

  const renderPlaceholderModal = () => {
    return (
      <div className="placeholder">
        <a href="https://www.fleetdm.com/learn-more-about/calendar-events">
          <img src={CalendarPreview} alt="Calendar preview" />
        </a>
        <div>
          To create calendar events for end users if their hosts fail policies,
          you must first connect Fleet to your Google Workspace service account.
        </div>
        <div>
          This can be configured in{" "}
          <b>Settings &gt; Integrations &gt; Calendars.</b>
        </div>
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
      </div>
    );
  };

  const renderConfiguredModal = () => (
    <div className={`${baseClass} form`}>
      <div className="form-header">
        <Slider
          value={formData.enabled}
          onChange={onFeatureEnabledChange}
          inactiveText="Disabled"
          activeText="Enabled"
        />
        <Button
          type="button"
          variant="text-link"
          onClick={() => {
            setSelectedPolicyToPreview(undefined);
            togglePreviewCalendarEvent();
          }}
        >
          Preview calendar event
        </Button>
      </div>
      <div
        className={`form ${formData.enabled ? "" : "form-fields--disabled"}`}
      >
        <InputField
          placeholder="https://server.com/example"
          label="Resolution webhook URL"
          onChange={onUrlChange}
          name="url"
          value={formData.url}
          error={formErrors.url}
          tooltip="Provide a URL to deliver a webhook request to."
          labelTooltipPosition="top-start"
          helpText="A request will be sent to this URL during the calendar event. Use it to trigger auto-remediation."
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
        {renderPolicies()}
      </div>
      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          onClick={onUpdatePolicyEnabledCalendarEvents}
          className="save-loading"
          isLoading={isUpdating}
          disabled={Object.keys(formErrors).length > 0}
        >
          Save
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );

  if (showPreviewCalendarEvent) {
    return (
      <CalendarEventPreviewModal
        onCancel={togglePreviewCalendarEvent}
        policy={selectedPolicyToPreview}
      />
    );
  }

  return (
    <Modal
      title="Calendar events"
      onExit={onExit}
      onEnter={
        configured
          ? () => {
              updatePolicyEnabledCalendarEvents(formData);
            }
          : onExit
      }
      className={baseClass}
      width="large"
    >
      {configured ? renderConfiguredModal() : renderPlaceholderModal()}
    </Modal>
  );
};

export default CalendarEventsModal;
