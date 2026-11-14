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
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import usePatchDeadlineSettings from "hooks/usePatchDeadlineSettings";
import { getErrorReason } from "interfaces/errors";
import {
  WEEKDAYS,
  WEEKDAY_ORDER,
  Weekday,
} from "interfaces/patch_deadline_settings";
import { ILoadTeamPoliciesResponse } from "interfaces/policy";
import patchDeadlineSettingsAPI from "services/entities/patch_deadline_settings";
import teamPoliciesAPI, {
  ITeamPoliciesQueryKey,
} from "services/entities/team_policies";

import { ILibrarySettingsCardProps } from "../../types";

import PatchDeadlineAppsTable from "./PatchDeadlineAppsTable";

const baseClass = "patch-deadline-section";

// Matches 24-hour HH:MM. Hours 00-23, minutes 00-59.
const TIME_PATTERN = /^([01]\d|2[0-3]):[0-5]\d$/;

interface IPatchDeadlineFormData {
  days: Weekday[];
  time: string;
}

const validate = (formData: IPatchDeadlineFormData): IFormErrors => {
  const errors: IFormErrors = {};
  if (formData.days.length === 0) errors.days = "Choose at least one day";
  if (!TIME_PATTERN.test(formData.time.trim())) {
    errors.time = "Enter time as HH:MM";
  }
  return errors;
};

const PatchDeadlineSection = ({
  router,
  fleetId,
  isRouteOk,
  canManage,
  gitOpsModeEnabled,
}: ILibrarySettingsCardProps) => {
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = usePatchDeadlineSettings(fleetId, {
    enabled: isRouteOk,
  });

  // Share the `teamPolicies` cache key family so other consumers of the same
  // (fleet, automation_type=patch) list dedupe through React Query instead of
  // double-fetching. Response is "all patch policies"; the filter to
  // `patch_deadline: true` happens client-side below.
  const appsQueryKey: ITeamPoliciesQueryKey = {
    scope: "teamPolicies",
    teamId: fleetId,
    automationType: "patch",
  };
  const { data: policiesData, isError: isAppsError } = useQuery<
    ILoadTeamPoliciesResponse,
    Error,
    ILoadTeamPoliciesResponse,
    [ITeamPoliciesQueryKey]
  >(
    [appsQueryKey],
    () =>
      teamPoliciesAPI.loadAllNew({
        teamId: fleetId,
        automationType: "patch",
      }),
    { enabled: isRouteOk, refetchOnWindowFocus: false }
  );
  // Patch-type policies with the Deadline option carry their software info
  // under `patch_software` (NOT `install_software`, which is for install-type
  // policies). Filter down to policies that have `patch_deadline: true` AND a
  // populated `patch_software` subobject.
  const apps = (policiesData?.policies ?? [])
    .filter((p) => p.patch_deadline && !!p.patch_software)
    .map((p) => p.patch_software!);
  const hasApps = apps.length > 0;

  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);

  const updateMutation = useMutation(patchDeadlineSettingsAPI.updateSettings, {
    onSuccess: () => {
      queryClient.invalidateQueries(["patchDeadlineSettings", fleetId]);
      notify.success("Successfully updated patch deadline.");
    },
    onError: (err: unknown) => {
      // Backend reports field-scoped errors as `{errors: [{name, reason}]}`.
      // The form shows those inline and toasts them; anything else gets a
      // generic toast.
      const fieldErrors: IFormErrors = {};
      const daysReason = getErrorReason(err, { nameEquals: "days" });
      const timeReason = getErrorReason(err, { nameEquals: "time" });
      if (daysReason) fieldErrors.days = daysReason;
      if (timeReason) fieldErrors.time = timeReason;
      if (Object.keys(fieldErrors).length > 0) {
        setServerErrors(fieldErrors);
      } else {
        notify.error("Couldn't update patch deadline.", { response: err });
      }
    },
  });

  const {
    formData,
    commitFields,
    getError,
    getTextFieldProps,
    handleSubmit,
    reset,
    isSubmitting,
  } = useFormValidation<IPatchDeadlineFormData>({
    initialFormData: { days: [], time: "" },
    validate,
    serverErrors,
    isSubmitting: updateMutation.isLoading,
  });

  // Reseed from the server whenever a new settings doc lands (first load, or
  // the fleet switcher pulled a different fleet's row).
  useEffect(() => {
    if (data) reset({ days: data.days, time: data.time });
  }, [data, reset]);

  const toggleDay = (key: Weekday) => {
    commitFields({
      days: formData.days.includes(key)
        ? formData.days.filter((d) => d !== key)
        : [...formData.days, key],
    });
  };

  const onValidSubmit = ({ days, time }: IPatchDeadlineFormData) => {
    // Canonical Mon-first order on the wire so the server (and the activity
    // feed renderer) never has to re-sort. Matches `WEEKDAY_ORDER` used by
    // the activity copy.
    const sortedDays = [...days].sort(
      (a, b) => WEEKDAY_ORDER[a] - WEEKDAY_ORDER[b]
    );
    updateMutation.mutate({ fleet_id: fleetId, days: sortedDays, time });
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

  // Role-based disable sticks (the user never gets to interact). Submit-time
  // disable uses readOnly so the field doesn't visually flicker to the greyed
  // "disabled" state for the few hundred ms of the PATCH round-trip.
  // GitOps mode also fully disables the fields — the Save button on its own
  // being blocked lets users edit values they can never persist.
  const roleDisabled = !canManage || gitOpsModeEnabled;
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
            Add a macOS Fleet-maintained app, and under <strong>Deploy</strong>{" "}
            select <strong>Patch</strong> &gt; <strong>Force patch</strong> &gt;{" "}
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

      {/*
        Native <form> so pressing Enter in Time submits (patterns.md 324-332).
        Apps subsection stays OUTSIDE the form so a stray Enter from a row
        can't trigger a settings save.
      */}
      <form
        onSubmit={handleSubmit(onValidSubmit)}
        className={`${baseClass}__form`}
      >
        <FormField
          name="patch-deadline-days"
          label="Days"
          error={getError("days")}
          disabled={roleDisabled}
        >
          <div className={`${baseClass}__days`}>
            {WEEKDAYS.map((w) => (
              <Checkbox
                key={w.key}
                name={w.key}
                ariaLabel={w.label}
                value={formData.days.includes(w.key)}
                onChange={() => toggleDay(w.key)}
                disabled={roleDisabled}
                readOnly={fieldReadOnly}
              >
                {w.label}
              </Checkbox>
            ))}
          </div>
        </FormField>

        <InputField
          {...getTextFieldProps("time")}
          label="Time"
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
                type="submit"
                disabled={disableChildren || isSubmitting}
                isLoading={isSubmitting}
              >
                Save
              </Button>
            )}
          />
        )}
      </form>

      {isAppsError && (
        <div>
          <SectionHeader title="Apps" />
          <DataError verticalPaddingSize="pad-large" />
        </div>
      )}
      {!isAppsError && hasApps && (
        <PatchDeadlineAppsTable fleetId={fleetId} apps={apps} router={router} />
      )}
    </div>
  );
};

export default PatchDeadlineSection;
