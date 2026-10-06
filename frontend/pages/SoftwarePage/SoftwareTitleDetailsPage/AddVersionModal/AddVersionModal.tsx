import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreApp } from "interfaces/software";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import VersionFormFields, {
  DEFAULT_VERSION_FORM_DATA,
  IVersionFormData,
} from "../VersionFormFields";

const baseClass = "add-version-modal";

interface IAddVersionModalProps {
  softwareTitleId: number;
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
  softwareTitleId,
  teamId,
  appStore,
  existingVersionNames,
  defaultTargetCustom = false,
  onExit,
  onSuccess,
}: IAddVersionModalProps) => {
  const queryClient = useQueryClient();

  const isIosOrIpados =
    appStore.platform === "ios" || appStore.platform === "ipados";
  // Platform-specific empty-config scaffolds. Mirrors EditConfigurationModal:
  // iOS/iPadOS gets a blank-line-between-tags XML plist; Android gets `{}`.
  // On submit, a value equal to the scaffold is treated as "no configuration"
  // and sent as undefined.
  const EMPTY_XML_SCAFFOLD = "<dict>\n  \n</dict>";
  const EMPTY_JSON_SCAFFOLD = "{}";
  const emptyScaffold = isIosOrIpados
    ? EMPTY_XML_SCAFFOLD
    : EMPTY_JSON_SCAFFOLD;

  const [formData, setFormData] = useState<IVersionFormData>({
    ...DEFAULT_VERSION_FORM_DATA,
    // Android has no self-service UI control; the API requires `self_service`,
    // so default to true to mirror the single-add Android flow.
    selfService: appStore.platform === "android",
    configuration: emptyScaffold,
    targetType: defaultTargetCustom ? "Custom" : "All hosts",
  });
  const [nameError, setNameError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Local to the modal so the form state in VersionFormFields stays mounted
  // (and unsaved input preserved) while the preview overlay is open.
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const existingNamesSet = new Set(
    existingVersionNames.map((n) => n.toLowerCase())
  );

  const validate = (candidate: string): string | null => {
    const trimmed = candidate.trim();
    if (!trimmed) return "Enter a version name";
    if (existingNamesSet.has(trimmed.toLowerCase())) {
      return "A version with this name already exists on this fleet";
    }
    return null;
  };

  const onFocusName = () => setNameError(null);
  const onBlurName = () => {
    if (formData.name) setNameError(validate(formData.name));
  };

  const onSubmit = async () => {
    const err = validate(formData.name);
    if (err) {
      setNameError(err);
      return;
    }

    setIsSubmitting(true);
    try {
      const labelsArray =
        formData.targetType === "Custom"
          ? buildLabelArray(formData.labelTargets)
          : undefined;

      await softwareAPI.addAppStoreAppVersion(teamId, {
        app_store_id: appStore.app_store_id,
        platform: appStore.platform,
        name: formData.name.trim(),
        self_service: formData.selfService,
        categories: formData.categories.length
          ? formData.categories
          : undefined,
        configuration:
          formData.configuration && formData.configuration !== emptyScaffold
            ? formData.configuration
            : undefined,
        labels_include_any:
          formData.customTarget === "labelsIncludeAny"
            ? labelsArray
            : undefined,
        labels_include_all:
          formData.customTarget === "labelsIncludeAll"
            ? labelsArray
            : undefined,
        labels_exclude_any:
          formData.customTarget === "labelsExcludeAny"
            ? labelsArray
            : undefined,
        auto_update_enabled: formData.autoUpdateEnabled || undefined,
        auto_update_window_start:
          formData.autoUpdateEnabled && formData.autoUpdateWindowStart
            ? formData.autoUpdateWindowStart
            : undefined,
        auto_update_window_end:
          formData.autoUpdateEnabled && formData.autoUpdateWindowEnd
            ? formData.autoUpdateWindowEnd
            : undefined,
      });

      notify.success(
        <>
          Successfully added new <b>{formData.name.trim()}</b> version.
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
        setNameError(reason);
      } else {
        notify.error("Couldn't add. Please try again.", { response: e });
      }
    }
    setIsSubmitting(false);
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
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
        >
          <VersionFormFields
            formData={formData}
            onChange={setFormData}
            platform={appStore.platform}
            appDisplayName={appStore.display_name || appStore.name}
            labels={labels ?? []}
            nameError={nameError}
            onNameFocus={onFocusName}
            onNameBlur={onBlurName}
            onClickPreviewEndUserExperience={
              appStore.platform !== "android"
                ? () => setShowPreviewEndUserExperience(true)
                : undefined
            }
          />

          <div className="modal-cta-wrap">
            <Button type="submit" isLoading={isSubmitting}>
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
          {/* softwareTitleId is kept in the modal's closure for the service
            call once per-version plumbing lands; the current add route targets
            the title via `app_store_id` + `platform`. */}
          <input
            type="hidden"
            name="software_title_id"
            value={softwareTitleId}
          />
        </form>
      </Modal>
      {showPreviewEndUserExperience && (
        <CategoriesEndUserExperienceModal
          onCancel={() => setShowPreviewEndUserExperience(false)}
          teamId={teamId}
          isIosOrIpadosApp={appStore.platform !== "android"}
        />
      )}
    </>
  );
};

export default AddVersionModal;
