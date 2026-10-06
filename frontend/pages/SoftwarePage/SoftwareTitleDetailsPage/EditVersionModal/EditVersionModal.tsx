import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreAppVersion } from "interfaces/software";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import softwareAPI from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import VersionFormFields, {
  IVersionFormData,
  versionToFormData,
} from "../VersionFormFields";

const baseClass = "edit-version-modal";

interface IEditVersionModalProps {
  softwareId: number;
  teamId: number;
  version: IAppStoreAppVersion;
  /** Other version names on this title, used for client-side uniqueness
   * validation. Excludes the version being edited. */
  siblingVersionNames: string[];
  onExit: () => void;
  onSuccess: () => void;
}

const buildLabelArray = (labelTargets: Record<string, boolean>): string[] =>
  Object.entries(labelTargets)
    .filter(([, selected]) => selected)
    .map(([name]) => name);

const EditVersionModal = ({
  softwareId,
  teamId,
  version,
  siblingVersionNames,
  onExit,
  onSuccess,
}: IEditVersionModalProps) => {
  const queryClient = useQueryClient();

  // Platform-specific empty-config scaffolds. Mirrors AddVersionModal and
  // EditConfigurationModal: iOS/iPadOS gets an empty XML plist, Android gets
  // `{}`. On submit, a value equal to the scaffold is treated as "no
  // configuration".
  const isIosOrIpados =
    version.platform === "ios" || version.platform === "ipados";
  const EMPTY_XML_SCAFFOLD = "<dict>\n  \n</dict>";
  const EMPTY_JSON_SCAFFOLD = "{}";
  const emptyScaffold = isIosOrIpados
    ? EMPTY_XML_SCAFFOLD
    : EMPTY_JSON_SCAFFOLD;

  const [formData, setFormData] = useState<IVersionFormData>(() => {
    const initial = versionToFormData(version);
    // Pre-fill the editor with the scaffold when the stored config is empty
    // so the user sees the right skeleton rather than a blank editor.
    if (!initial.configuration) {
      initial.configuration = emptyScaffold;
    }
    return initial;
  });
  const [nameError, setNameError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Local to the modal so the form state in VersionFormFields stays mounted
  // (and unsaved edits preserved) while the preview overlay is open.
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  const siblingNamesSet = new Set(
    siblingVersionNames.map((n) => n.toLowerCase())
  );

  const validate = (candidate: string): string | null => {
    const trimmed = candidate.trim();
    if (!trimmed) return "Enter a version name";
    if (siblingNamesSet.has(trimmed.toLowerCase())) {
      return "A version with this name already exists on this fleet";
    }
    return null;
  };

  const onFocusName = () => setNameError(null);
  const onBlurName = () => {
    if (formData.name !== version.name) setNameError(validate(formData.name));
  };

  const onSubmit = async () => {
    const err = validate(formData.name);
    if (err) {
      setNameError(err);
      return;
    }

    setIsSubmitting(true);
    try {
      // Target → always send the three label arrays so the backend can
      // normalize. "All hosts" sends empty arrays to clear any existing
      // label scope. "Custom" sends the active list on the active key and
      // empty arrays on the other two.
      const activeLabels =
        formData.targetType === "Custom"
          ? buildLabelArray(formData.labelTargets)
          : [];
      const labelsIncludeAny =
        formData.targetType === "Custom" &&
        formData.customTarget === "labelsIncludeAny"
          ? activeLabels
          : [];
      const labelsIncludeAll =
        formData.targetType === "Custom" &&
        formData.customTarget === "labelsIncludeAll"
          ? activeLabels
          : [];
      const labelsExcludeAny =
        formData.targetType === "Custom" &&
        formData.customTarget === "labelsExcludeAny"
          ? activeLabels
          : [];

      await softwareAPI.editAppStoreAppVersion(softwareId, teamId, version.id, {
        name: formData.name.trim(),
        self_service: formData.selfService,
        categories: formData.categories,
        configuration:
          formData.configuration && formData.configuration !== emptyScaffold
            ? formData.configuration
            : "",
        labels_include_any: labelsIncludeAny,
        labels_include_all: labelsIncludeAll,
        labels_exclude_any: labelsExcludeAny,
        auto_update_enabled: formData.autoUpdateEnabled,
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
          Successfully edited <b>{formData.name.trim()}</b>.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
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
        notify.error("Couldn't edit. Please try again.", { response: e });
      }
    }
    setIsSubmitting(false);
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
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
        >
          <VersionFormFields
            formData={formData}
            onChange={setFormData}
            platform={version.platform}
            appDisplayName={version.display_name || version.name}
            labels={labels ?? []}
            nameError={nameError}
            onNameFocus={onFocusName}
            onNameBlur={onBlurName}
            onClickPreviewEndUserExperience={
              version.platform !== "android"
                ? () => setShowPreviewEndUserExperience(true)
                : undefined
            }
          />

          <div className="modal-cta-wrap">
            <Button type="submit" isLoading={isSubmitting}>
              Save
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
          isIosOrIpadosApp={version.platform !== "android"}
        />
      )}
    </>
  );
};

export default EditVersionModal;
