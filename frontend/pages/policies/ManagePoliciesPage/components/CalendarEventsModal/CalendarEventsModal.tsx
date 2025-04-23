import React, { useState, useRef, useContext } from "react";
import { noop } from "lodash";

import { AppContext } from "context/app";
import { syntaxHighlight } from "utilities/helpers";
import classnames from "classnames";
import validURL from "components/forms/validators/valid_url";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Icon from "components/Icon";
import { IPaginatedListHandle } from "components/PaginatedList";
import CalendarEventPreviewModal from "../CalendarEventPreviewModal";
import CalendarPreview from "../../../../../../assets/images/calendar-preview-720x436@2x.png";
import PoliciesPaginatedList, {
  IFormPolicy,
} from "../PoliciesPaginatedList/PoliciesPaginatedList";

const baseClass = "calendar-events-modal";

export interface ICalendarEventsFormData {
  enabled: boolean;
  url: string;
  changedPolicies: IFormPolicy[];
}

interface ICalendarEventsModal {
  onExit: () => void;
  onSubmit: (formData: ICalendarEventsFormData) => void;
  isUpdating: boolean;
  configured: boolean;
  enabled: boolean;
  url: string;
  teamId: number;
  gitOpsModeEnabled?: boolean;
}

const CalendarEventsModal = ({
  onExit,
  onSubmit,
  isUpdating,
  configured,
  enabled,
  url,
  teamId,
  gitOpsModeEnabled = false,
}: ICalendarEventsModal) => {
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);
  const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);

  const isAdmin = isGlobalAdmin || isTeamAdmin;

  const [formData, setFormData] = useState<ICalendarEventsFormData>({
    enabled,
    url,
    changedPolicies: [],
  });

  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showPreviewCalendarEvent, setShowPreviewCalendarEvent] = useState(
    false
  );
  const [showExamplePayload, setShowExamplePayload] = useState(false);
  const [selectedPolicyToPreview, setSelectedPolicyToPreview] = useState<
    IFormPolicy | undefined
  >();

  // Used on URL change only when URL error exists and always on attempting to save
  const validateForm = (newFormData: ICalendarEventsFormData) => {
    const errors: Record<string, string> = {};
    const { url: newUrl } = newFormData;
    if (
      newFormData.enabled &&
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

  const onUpdateCalendarEvents = () => {
    const errors = validateForm(formData);

    if (Object.keys(errors).length > 0) {
      setFormErrors(errors);
    } else if (paginatedListRef.current) {
      const changedPolicies = paginatedListRef.current.getDirtyItems();
      onSubmit({ ...formData, changedPolicies });
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
          This can be configured in <b>Settings</b> &gt; <b>Integrations</b>{" "}
          &gt; <b>Calendars.</b>
        </div>
        <CustomLink
          url="https://www.fleetdm.com/learn-more-about/calendar-events"
          text="Learn more"
          newTab
        />
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </div>
    );
  };

  const renderAdminHeader = () => (
    <div className="form-header">
      <Slider
        value={formData.enabled}
        onChange={onFeatureEnabledChange}
        inactiveText="Disabled"
        activeText="Enabled"
        disabled={gitOpsModeEnabled}
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
  );

  /** Maintainer does not have access to update calendar event
  / configuration only to set the automated policies
  / Modal not available for maintainers if calendar is disabled but
  / disabled inputs here as fail safe and to match admin view.
  */
  const renderMaintainerHeader = () => (
    <>
      <div className="form-header">
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
      {showExamplePayload && renderExamplePayload()}
    </>
  );

  const renderConfiguredModal = () => (
    <div className={`${baseClass}__configured-modal form`}>
      {isAdmin ? renderAdminHeader() : renderMaintainerHeader()}
      <div
        className={`form ${formData.enabled ? "" : "form-fields--disabled"}`}
      >
        {isAdmin && (
          <>
            <InputField
              placeholder="https://server.com/example"
              label="Resolution webhook URL"
              onChange={onUrlChange}
              name="url"
              value={formData.url}
              error={formErrors.url}
              tooltip="Provide a URL to deliver a webhook request to."
              helpText="A request will be sent to this URL during the calendar event. Use it to trigger auto-remediation."
              disabled={!formData.enabled || gitOpsModeEnabled}
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
              disabled={!formData.enabled || gitOpsModeEnabled}
            />
            {showExamplePayload && renderExamplePayload()}
          </>
        )}
      </div>
      <div>
        <PoliciesPaginatedList
          ref={paginatedListRef}
          isSelected="calendar_events_enabled"
          onToggleItem={(item: IFormPolicy) => {
            item.calendar_events_enabled = !item.calendar_events_enabled;
            return item;
          }}
          renderItemRow={(item: IFormPolicy) => {
            return (
              <Button
                variant="text-icon"
                onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                  e.stopPropagation();
                  setSelectedPolicyToPreview(item);
                  togglePreviewCalendarEvent();
                }}
                className="policy-row__preview-button"
              >
                <Icon name="eye" /> Preview
              </Button>
            );
          }}
          footer={
            <>
              A calendar event will be created for end users if one of their
              hosts fail any of these policies.{" "}
              <CustomLink
                url="https://www.fleetdm.com/learn-more-about/calendar-events"
                text="Learn more"
                newTab
                disableKeyboardNavigation={!formData.enabled}
              />
            </>
          }
          isUpdating={isUpdating}
          onSubmit={onUpdateCalendarEvents}
          onCancel={onExit}
          teamId={teamId}
          disabled={!formData.enabled}
        />
      </div>
    </div>
  );

  const classes = classnames(baseClass, {
    [`${baseClass}__hide-main`]: showPreviewCalendarEvent,
  });
  return (
    <>
      <Modal
        title="Calendar events"
        // Disable exit when preview modal is open, so that escape key
        // only closes the preview.
        onExit={showPreviewCalendarEvent ? noop : onExit}
        onEnter={
          configured
            ? () => {
                onUpdateCalendarEvents();
              }
            : onExit
        }
        className={classes}
        width="large"
      >
        {configured ? renderConfiguredModal() : renderPlaceholderModal()}
      </Modal>
      {showPreviewCalendarEvent && (
        <CalendarEventPreviewModal
          onCancel={togglePreviewCalendarEvent}
          policy={selectedPolicyToPreview}
        />
      )}
    </>
  );
};

export default CalendarEventsModal;
