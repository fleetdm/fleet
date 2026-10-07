import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreApp } from "interfaces/software";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import VersionFormFields, {
  DEFAULT_VERSION_FORM_DATA,
  getEmptyConfigScaffold,
  IVersionFormData,
  validateVersionForm,
} from "../VersionFormFields";

const baseClass = "add-version-modal";

interface IAddVersionModalProps {
  teamId: number;
  /** The title's back-compat `app_store_app` envelope. The new version
   * inherits `app_store_id` and `platform` from this. */
  appStore: IAppStoreApp;
  /** Existing version names on this title, used for client-side uniqueness
   * validation before the request hits the backend. */
  existingVersionNames: string[];
  /** When true, the "Target" field defaults to `Custom` so the admin's label
   * scope wins the first-added race. Set when adding the 2nd+ version. */
  defaultTargetCustom?: boolean;
  onExit: () => void;
  onSuccess: () => void;
}

const buildLabelArray = (
  labelTargets: Record<string, boolean>
): string[] | undefined => {
  const names = Object.entries(labelTargets)
    .filter(([, selected]) => selected)
    .map(([name]) => name);
  return names.length ? names : undefined;
};

const AddVersionModal = ({
  teamId,
  appStore,
  existingVersionNames,
  defaultTargetCustom = false,
  onExit,
  onSuccess,
}: IAddVersionModalProps) => {
  const queryClient = useQueryClient();

  // Submitting an unmodified scaffold is treated as "no configuration".
  const emptyScaffold = getEmptyConfigScaffold(appStore.platform);

  const existingNamesSet = new Set(
    existingVersionNames.map((n) => n.toLowerCase())
  );

  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const validate = (data: IVersionFormData): IFormErrors =>
    validateVersionForm(
      data,
      (trimmed) => !existingNamesSet.has(trimmed.toLowerCase()),
      appStore.platform
    );

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
    initialFormData: {
      ...DEFAULT_VERSION_FORM_DATA,
      // Android: no UI control, API requires `self_service` → default true.
      selfService: appStore.platform === "android",
      configuration: emptyScaffold,
      targetType: defaultTargetCustom ? "Custom" : "All hosts",
    },
    validate,
    serverErrors,
  });

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const onValidSubmit = async (data: IVersionFormData) => {
    const labelsArray =
      data.targetType === "Custom"
        ? buildLabelArray(data.labelTargets)
        : undefined;

    // Android configuration is a JSON object on the wire; parse the editor
    // string before sending. iOS/iPadOS send the XML plist as a string.
    const hasConfig =
      !!data.configuration && data.configuration !== emptyScaffold;
    let configurationPayload: string | Record<string, unknown> | undefined;
    if (hasConfig) {
      configurationPayload =
        appStore.platform === "android"
          ? (JSON.parse(data.configuration) as Record<string, unknown>)
          : data.configuration;
    }

    try {
      await softwareAPI.addAppStoreAppVersion(teamId, {
        app_store_id: appStore.app_store_id,
        platform: appStore.platform,
        name: data.name,
        self_service: data.selfService,
        categories: data.categories.length ? data.categories : undefined,
        configuration: configurationPayload,
        labels_include_any:
          data.customTarget === "labelsIncludeAny" ? labelsArray : undefined,
        labels_include_all:
          data.customTarget === "labelsIncludeAll" ? labelsArray : undefined,
        labels_exclude_any:
          data.customTarget === "labelsExcludeAny" ? labelsArray : undefined,
        auto_update_enabled: data.autoUpdateEnabled || undefined,
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
          Successfully added new <strong>{data.name}</strong> version.
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
      if (reason?.toLowerCase().includes("name")) {
        setServerErrors({ name: reason });
      } else {
        notify.error("Couldn't add. Please try again.", { response: e });
      }
    }
  };

  return (
    <>
      <Modal
        className={baseClass}
        title="Add version"
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
            platform={appStore.platform}
            appDisplayName={appStore.display_name || appStore.name}
            labels={labels ?? []}
            teamId={teamId}
            onClickPreviewEndUserExperience={() =>
              setShowPreviewEndUserExperience(true)
            }
          />

          <div className="modal-cta-wrap">
            <Button
              type="submit"
              isLoading={isSubmitting}
              disabled={isSubmitting}
            >
              Add
            </Button>
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

export default AddVersionModal;
