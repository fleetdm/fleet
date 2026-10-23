import React, { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import FormField from "components/forms/FormField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PageDescription from "components/PageDescription";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import {
  IPatchDeadlineAppsResponse,
  IPatchDeadlineSettings,
  Weekday,
} from "interfaces/patch_deadline_settings";
import patchDeadlineSettingsAPI from "services/entities/patch_deadline_settings";

import { ILibrarySettingsCardProps } from "../../types";

import PatchDeadlineAppsTable from "./PatchDeadlineAppsTable";

const baseClass = "patch-deadline-section";

const WEEKDAYS: ReadonlyArray<{ key: Weekday; label: string }> = [
  { key: "monday", label: "Monday" },
  { key: "tuesday", label: "Tuesday" },
  { key: "wednesday", label: "Wednesday" },
  { key: "thursday", label: "Thursday" },
  { key: "friday", label: "Friday" },
  { key: "saturday", label: "Saturday" },
  { key: "sunday", label: "Sunday" },
];

// Matches 24-hour HH:MM. Hours 00-23, minutes 00-59.
const TIME_PATTERN = /^([01]\d|2[0-3]):[0-5]\d$/;

interface IFieldErrors {
  days?: string;
  time?: string;
}

const PatchDeadlineSection = ({
  router,
  fleetId,
  isRouteOk,
  canManage,
}: ILibrarySettingsCardProps) => {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery<IPatchDeadlineSettings, Error>(
    ["patchDeadlineSettings", fleetId],
    () => patchDeadlineSettingsAPI.getSettings(fleetId),
    { enabled: isRouteOk, refetchOnWindowFocus: false }
  );

  // The Apps subsection and the "Apps with this option are listed below."
  // pointer in the description are both gated on this query having a non-empty
  // result — hide both when the fleet has no deadline apps.
  const { data: appsData } = useQuery<IPatchDeadlineAppsResponse, Error>(
    ["patchDeadlineApps", fleetId],
    () => patchDeadlineSettingsAPI.getApps(fleetId),
    { enabled: isRouteOk, refetchOnWindowFocus: false }
  );
  const apps = appsData?.apps ?? [];
  const hasApps = apps.length > 0;

  const [days, setDays] = useState<Weekday[]>([]);
  const [time, setTime] = useState("");
  const [errors, setErrors] = useState<IFieldErrors>({});

  // Reseed from the server whenever a new settings doc lands (first load, or
  // the fleet switcher pulled a different fleet's row).
  useEffect(() => {
    if (!data) return;
    setDays(data.days);
    setTime(data.time);
    setErrors({});
  }, [data]);

  const validate = (nextDays: Weekday[], nextTime: string): IFieldErrors => {
    const next: IFieldErrors = {};
    if (nextDays.length === 0) next.days = "Pick at least one day";
    if (!TIME_PATTERN.test(nextTime)) next.time = "Enter time as HH:MM";
    return next;
  };

  const toggleDay = (key: Weekday) => {
    const nextDays = days.includes(key)
      ? days.filter((d) => d !== key)
      : [...days, key];
    setDays(nextDays);
    // Any interaction clears a shown error (Fleet validation rule: errors
    // never (re)appear on keystroke/toggle — only on blur and submit).
    setErrors((prev) => ({ ...prev, days: undefined }));
  };

  const updateMutation = useMutation(patchDeadlineSettingsAPI.updateSettings, {
    onSuccess: () => {
      queryClient.invalidateQueries(["patchDeadlineSettings", fleetId]);
      notify.success("Successfully updated patch deadline.");
    },
    onError: (err: unknown) => {
      // Backend validates days and time and reports field-scoped errors as
      // `{errors: [{name, reason}]}`. Surface them inline next to the field,
      // and keep the toast so systemic errors (500s with no `errors`) still
      // surface for the user.
      const daysReason = getErrorReason(err, { nameEquals: "days" });
      const timeReason = getErrorReason(err, { nameEquals: "time" });
      if (daysReason || timeReason) {
        setErrors({
          days: daysReason || undefined,
          time: timeReason || undefined,
        });
      }
      notify.error("Couldn't update patch deadline.", { response: err });
    },
  });

  const onSave = () => {
    const nextErrors = validate(days, time);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length > 0) return;
    updateMutation.mutate({ fleet_id: fleetId, days, time });
  };

  if (!isRouteOk || isLoading) {
    return (
      <div className={baseClass}>
        <Spinner />
      </div>
    );
  }

  if (isError) {
    return (
      <div className={baseClass}>
        <DataError verticalPaddingSize="pad-xxxlarge" selfCenter />
      </div>
    );
  }

  const isSubmitting = updateMutation.isLoading;
  // Role-based disable sticks (the user never gets to interact). Submit-time
  // disable uses readOnly so the field doesn't visually flicker to the greyed
  // "disabled" state for the few hundred ms of the PATCH round-trip.
  const roleDisabled = !canManage;
  const fieldReadOnly = isSubmitting;

  return (
    <div className={baseClass}>
      <SectionHeader
        wrapperCustomClass={`${baseClass}__header`}
        title="Patch deadline"
      />
      <PageDescription
        variant="tab-panel"
        content={
          <>
            Add a Fleet-maintained app, and under <strong>Deploy</strong> select{" "}
            <strong>Patch</strong> &gt; <strong>Force patch</strong> &gt;{" "}
            <strong>Deadline</strong>.{" "}
            {hasApps && <>Apps with this option are listed below. </>}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/patching-end-user-experience"
              text="See end user experience"
              newTab
            />
          </>
        }
      />

      <FormField name="patch-deadline-days" label="Days" error={errors.days}>
        <div className={`${baseClass}__days`}>
          {WEEKDAYS.map((w) => (
            <Checkbox
              key={w.key}
              name={w.key}
              ariaLabel={w.label}
              value={days.includes(w.key)}
              onChange={() => toggleDay(w.key)}
              disabled={roleDisabled || fieldReadOnly}
            >
              {w.label}
            </Checkbox>
          ))}
        </div>
      </FormField>

      <InputField
        label="Time"
        name="patch-deadline-time"
        value={time}
        onChange={(v: string) => setTime(v)}
        onBlur={() => {
          const next = validate(days, time);
          setErrors((prev) => ({ ...prev, time: next.time }));
        }}
        onFocus={() => setErrors((prev) => ({ ...prev, time: undefined }))}
        error={errors.time}
        helpText={`Time is formatted as HH:MM in 24-hour format (e.g. "17:00"). All hosts' local time.`}
        disabled={roleDisabled}
        readOnly={fieldReadOnly}
        inputOptions={{ maxLength: 5 }}
      />

      {canManage && (
        <GitOpsModeTooltipWrapper
          entityType="software"
          position="top"
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <Button
              onClick={onSave}
              disabled={disableChildren || isSubmitting}
              isLoading={isSubmitting}
            >
              Save
            </Button>
          )}
        />
      )}

      {hasApps && (
        <PatchDeadlineAppsTable fleetId={fleetId} apps={apps} router={router} />
      )}
    </div>
  );
};

export default PatchDeadlineSection;
