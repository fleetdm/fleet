import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreAppVersion } from "interfaces/software";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import { buildSelectedLabelsArray } from "pages/SoftwarePage/helpers";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import VersionFormFields, {
  getEmptyConfigScaffold,
  IVersionFormData,
  validateVersionForm,
  versionToFormData,
} from "../VersionFormFields";

const baseClass = "edit-version-modal";

interface IEditVersionModalProps {
  softwareId: number;
  teamId: number;
  version: IAppStoreAppVersion;
  /** Resolved title display name — `version.name` is the admin's version
   * label (e.g. "Production"), not the app name, so we can't fall back to
   * it for app-identity copy like the Auto updates help text. */
  titleDisplayName: string;
  /** Other version names on this title, used for client-side uniqueness
   * validation. Excludes the version being edited. */
  siblingVersionNames: string[];
  onExit: () => void;
  onSuccess: () => void;
}

const EditVersionModal = ({
  softwareId,
  teamId,
  version,
  titleDisplayName,
  siblingVersionNames,
  onExit,
  onSuccess,
}: IEditVersionModalProps) => {
  const queryClient = useQueryClient();

  // Submitting an unmodified scaffold is treated as "no configuration".
  const emptyScaffold = getEmptyConfigScaffold(version.platform);

  const siblingNamesSet = new Set(
    siblingVersionNames.map((n) => n.toLowerCase())
  );

  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const validate = (data: IVersionFormData): IFormErrors =>
    validateVersionForm(
      data,
      (trimmed) => !siblingNamesSet.has(trimmed.toLowerCase()),
      version.platform
    );

  const initialFormData = (() => {
    const base = versionToFormData(version);
    // Pre-fill the editor with the scaffold when the stored config is empty.
    if (!base.configuration) base.configuration = emptyScaffold;
    return base;
  })();

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IVersionFormData>({
    initialFormData,
    validate,
    serverErrors,
  });

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const onValidSubmit = async (data: IVersionFormData) => {
    // Always send all three label arrays so the backend can normalize.
    // "All hosts" clears every scope with empty arrays; "Custom" fills the
    // active key and empties the other two.
    const activeLabels =
      data.targetType === "Custom"
        ? buildSelectedLabelsArray(data.labelTargets)
        : [];
    const labelsIncludeAny =
      data.targetType === "Custom" && data.customTarget === "labelsIncludeAny"
        ? activeLabels
        : [];
    const labelsIncludeAll =
      data.targetType === "Custom" && data.customTarget === "labelsIncludeAll"
        ? activeLabels
        : [];
    const labelsExcludeAny =
      data.targetType === "Custom" && data.customTarget === "labelsExcludeAny"
        ? activeLabels
        : [];

    // Empty editor or an unmodified scaffold both clear the config (null for
    // iOS/iPadOS, {} for Android, matching the backend's clear values). Any
    // other content is a set/update. Typing `{}` into the Android editor is a
    // deliberate clear even though it matches the scaffold.
    let configurationPayload: string | Record<string, unknown> | null;
    if (!data.configuration || data.configuration === emptyScaffold) {
      configurationPayload = version.platform === "android" ? {} : null;
    } else {
      configurationPayload =
        version.platform === "android"
          ? (JSON.parse(data.configuration) as Record<string, unknown>)
          : data.configuration;
    }

    try {
      await softwareAPI.editAppStoreAppVersion(softwareId, teamId, version.id, {
        name: data.name,
        self_service: data.selfService,
        categories: data.categories,
        configuration: configurationPayload,
        labels_include_any: labelsIncludeAny,
        labels_include_all: labelsIncludeAll,
        labels_exclude_any: labelsExcludeAny,
        auto_update_enabled: data.autoUpdateEnabled,
        auto_update_window_start:
          data.autoUpdateEnabled && data.autoUpdateWindowStart
            ? data.autoUpdateWindowStart
            : undefined,
        auto_update_window_end:
          data.autoUpdateEnabled && data.autoUpdateWindowEnd
            ? data.autoUpdateWindowEnd
            : undefined,
      });

      notify.success(
        <>
          Successfully edited <strong>{data.name}</strong>.
        </>
      );
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-titles" }],
      });
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-library" }],
      });
      onSuccess();
    } catch (e) {
      const reason = getErrorReason(e);
      // Only route the backend's duplicate-version-name conflict to the Name
      // field. Any other error containing "name" (e.g. an "Unsupported
      // variable $FLEET_VAR_..._USERNAME" from a configuration variable) stays
      // in the error toast so admins see the real reason.
      if (reason?.toLowerCase().includes("a version named")) {
        setServerErrors({ name: reason });
      } else {
        notify.error("Couldn't edit. Please try again.", { response: e });
      }
    }
  };

  return (
    <>
      <Modal
        className={baseClass}
        title="Edit version"
        onExit={onExit}
        width="large"
      >
        <form
          className={`${baseClass}__form`}
          onSubmit={handleSubmit(onValidSubmit)}
        >
          <VersionFormFields
            formData={formData}
            setField={setField}
            commitFields={commitFields}
            getError={getError}
            clearFieldError={clearFieldError}
            validateField={validateField}
            platform={version.platform}
            appDisplayName={titleDisplayName}
            labels={labels ?? []}
            teamId={teamId}
            onClickPreviewEndUserExperience={() =>
              setShowPreviewEndUserExperience(true)
            }
          />

          <div className="modal-cta-wrap">
            <GitOpsModeTooltipWrapper
              position="top"
              tipOffset={8}
              entityType="software"
              renderChildren={(gitOpsDisabled) => (
                <Button
                  type="submit"
                  isLoading={isSubmitting}
                  disabled={isSubmitting || !!gitOpsDisabled}
                >
                  Save
                </Button>
              )}
            />
            <Button
              onClick={onExit}
              variant="secondary"
              disabled={isSubmitting}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Modal>
      {showPreviewEndUserExperience && (
        <CategoriesEndUserExperienceModal
          onCancel={() => setShowPreviewEndUserExperience(false)}
          teamId={teamId}
          // Button that opens this is iOS/iPadOS-only (gated in VersionFormFields).
          isIosOrIpadosApp
        />
      )}
    </>
  );
};

export default EditVersionModal;
