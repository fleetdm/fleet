import React, {
  forwardRef,
  useContext,
  useImperativeHandle,
  useState,
} from "react";
import { AppContext } from "context/app";
import { syntaxHighlight } from "utilities/helpers";
import validURL from "components/forms/validators/valid_url";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import Slider from "components/forms/fields/Slider";
import InputField from "components/forms/fields/InputField";
import paths from "router/paths";
import InfoBanner from "components/InfoBanner/InfoBanner";

const baseClass = "calendar-events-modal";

export interface ICalendarEventsModalData {
  enabled: boolean;
  url: string;
}

export interface ICalendarEventsModalHandle {
  getFormData: () => ICalendarEventsModalData | null;
  validate: () => boolean;
  isDirty: () => boolean;
}

interface ICalendarEventsModalProps {
  configured: boolean;
  enabled: boolean;
  url: string;
  gitOpsModeEnabled?: boolean;
}

const CalendarEventsModal = forwardRef<
  ICalendarEventsModalHandle,
  ICalendarEventsModalProps
>(
  (
    {
      configured,
      enabled,
      url,
      gitOpsModeEnabled = false,
    }: ICalendarEventsModalProps,
    ref
  ) => {
    const { isGlobalAdmin, isTeamAdmin } = useContext(AppContext);
    const isAdmin = isGlobalAdmin || isTeamAdmin;

    const [formData, setFormData] = useState<ICalendarEventsModalData>({
      enabled,
      url,
    });

    const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
      {}
    );
    const [showExamplePayload, setShowExamplePayload] = useState(false);

    const validateForm = (newFormData: ICalendarEventsModalData) => {
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

    useImperativeHandle(ref, () => ({
      getFormData: () => (configured ? formData : null),
      validate: () => {
        if (!configured) return true;
        const errors = validateForm(formData);
        setFormErrors(errors);
        return Object.keys(errors).length === 0;
      },
      isDirty: () =>
        configured && (formData.enabled !== enabled || formData.url !== url),
    }));

    const onFeatureEnabledChange = () => {
      const newFormData = { ...formData, enabled: !formData.enabled };

      const isDisabling = newFormData.enabled === false;

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
      if (formErrors.url) {
        setFormErrors(validateForm(newFormData));
      }
      setFormData(newFormData);
    };

    const renderExamplePayload = () => (
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

    return (
      <>
        <div className={`${baseClass} form`}>
          <div className={`${baseClass}__header`}>
            <p className={`${baseClass}__description`}>
              Schedule maintenance windows for end users failing policies.{" "}
              <CustomLink
                url="https://www.fleetdm.com/learn-more-about/calendar-events"
                text="Learn more"
                newTab
              />
            </p>
          </div>
          {!configured && (
            <InfoBanner className={baseClass}>
              To use calendar automations, connect Fleet to Google Workspace in{" "}
              <CustomLink
                url={paths.ADMIN_INTEGRATIONS_CALENDARS}
                text="Settings &gt; Integrations &gt; Calendars"
                multiline
              />
              .
            </InfoBanner>
          )}
          {configured && (
            <>
              <Slider
                value={formData.enabled}
                onChange={onFeatureEnabledChange}
                inactiveText="Disabled"
                activeText="Enabled"
                disabled={gitOpsModeEnabled || !isAdmin}
              />
              {isAdmin && (
                <div
                  className={`form ${
                    formData.enabled ? "" : "form-fields--disabled"
                  }`}
                >
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
                    onClick={() => setShowExamplePayload(!showExamplePayload)}
                    disabled={!formData.enabled || gitOpsModeEnabled}
                  />
                  {showExamplePayload && renderExamplePayload()}
                </div>
              )}
            </>
          )}
        </div>
      </>
    );
  }
);

CalendarEventsModal.displayName = "CalendarEventsModal";

export default CalendarEventsModal;
