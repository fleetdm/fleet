import React from "react";

import CustomLink from "components/CustomLink";
import Editor from "components/Editor";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import Slider from "components/forms/fields/Slider";
import FormField from "components/forms/FormField";
import InfoBanner from "components/InfoBanner";
import { DropdownTargetLabelSelector } from "components/TargetLabelSelector";
import { ILabelSummary } from "interfaces/label";
import { IAppStoreAppVersion, SoftwareCategory } from "interfaces/software";
import { CategoriesSelector } from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector/SoftwareOptionsSelector";
import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
  getAutoUpdateWindowDurationMinutes,
  HHMM_RE,
} from "pages/SoftwarePage/helpers";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "version-form-fields";

export type VersionTargetType = "All hosts" | "Custom";
export type VersionCustomTarget =
  | "labelsIncludeAny"
  | "labelsIncludeAll"
  | "labelsExcludeAny";

export interface IVersionFormData {
  name: string;
  configuration: string;
  selfService: boolean;
  categories: SoftwareCategory[];
  targetType: VersionTargetType;
  customTarget: VersionCustomTarget;
  labelTargets: Record<string, boolean>;
  autoUpdateEnabled: boolean;
  autoUpdateWindowStart: string;
  autoUpdateWindowEnd: string;
}

// Scaffolds pre-filled into the configuration Editor on a fresh form. The
// Add/Edit modals treat an unmodified scaffold as "no configuration" so the
// row isn't force-saved with an empty value.
export const EMPTY_XML_SCAFFOLD = "<dict>\n  \n</dict>";
export const EMPTY_JSON_SCAFFOLD = "{}";
export const getEmptyConfigScaffold = (platform: string): string =>
  platform === "ios" || platform === "ipados"
    ? EMPTY_XML_SCAFFOLD
    : EMPTY_JSON_SCAFFOLD;

export const DEFAULT_VERSION_FORM_DATA: IVersionFormData = {
  name: "",
  configuration: "",
  selfService: false,
  categories: [],
  targetType: "All hosts",
  customTarget: "labelsIncludeAny",
  labelTargets: {},
  autoUpdateEnabled: false,
  autoUpdateWindowStart: "",
  autoUpdateWindowEnd: "",
};

/** Shared validator for Add and Edit version modals. Mirrors the shape used
 * by `EditAutoUpdateConfigModal/helpers.tsx`: at least one label when Target
 * is Custom, and a valid >=60-minute window (or wrap-to-next-day) when
 * Auto updates is enabled. Name validation is caller-supplied so the sibling
 * set differs per modal. */
export const validateVersionForm = (
  data: IVersionFormData,
  checkNameUniqueness: (trimmedName: string) => boolean,
  platform: string
): Record<string, string> => {
  const errors: Record<string, string> = {};
  const trimmed = data.name.trim();
  if (!trimmed) {
    errors.name = "Enter a version name";
  } else if (!checkNameUniqueness(trimmed)) {
    errors.name = "A version with this name already exists on this fleet";
  }

  // Android configuration must be a valid JSON object (the API strict-decodes
  // it). iOS/iPadOS XML validity isn't checked client-side; the backend
  // rejects malformed plists with a readable error.
  if (platform === "android" && data.configuration) {
    try {
      JSON.parse(data.configuration);
    } catch {
      errors.configuration = "Enter valid JSON";
    }
  }

  if (data.targetType === "Custom") {
    const selected = Object.values(data.labelTargets).filter(Boolean).length;
    if (selected === 0) {
      errors.labelTargets = "Select at least one label";
    }
  }

  if (data.autoUpdateEnabled) {
    const startValid =
      !!data.autoUpdateWindowStart && HHMM_RE.test(data.autoUpdateWindowStart);
    const endValid =
      !!data.autoUpdateWindowEnd && HHMM_RE.test(data.autoUpdateWindowEnd);
    if (!startValid) {
      errors.autoUpdateWindowStart = data.autoUpdateWindowStart
        ? "Use HH:MM format"
        : "Enter a start time";
    }
    if (!endValid) {
      errors.autoUpdateWindowEnd = data.autoUpdateWindowEnd
        ? "Use HH:MM format"
        : "Enter an end time";
    }
    if (startValid && endValid) {
      const durationMinutes = getAutoUpdateWindowDurationMinutes(
        data.autoUpdateWindowStart,
        data.autoUpdateWindowEnd
      );
      if (durationMinutes !== null && durationMinutes < 60) {
        errors.autoUpdateWindowEnd = "Window must be at least 60 minutes";
      }
    }
  }

  return errors;
};

interface IResolvedTarget {
  customTarget: VersionCustomTarget;
  targetType: VersionTargetType;
  labelTargets: Record<string, boolean>;
}

const toLabelTargets = (labels: { name: string }[]): Record<string, boolean> =>
  labels.reduce<Record<string, boolean>>((acc, l) => {
    acc[l.name] = true;
    return acc;
  }, {});

const resolveTargetFromVersion = (
  version: IAppStoreAppVersion
): IResolvedTarget => {
  if (version.labels_include_all?.length) {
    return {
      customTarget: "labelsIncludeAll",
      targetType: "Custom",
      labelTargets: toLabelTargets(version.labels_include_all),
    };
  }
  if (version.labels_exclude_any?.length) {
    return {
      customTarget: "labelsExcludeAny",
      targetType: "Custom",
      labelTargets: toLabelTargets(version.labels_exclude_any),
    };
  }
  if (version.labels_include_any?.length) {
    return {
      customTarget: "labelsIncludeAny",
      targetType: "Custom",
      labelTargets: toLabelTargets(version.labels_include_any),
    };
  }
  return {
    customTarget: "labelsIncludeAny",
    targetType: "All hosts",
    labelTargets: {},
  };
};

/** Translate a persisted version into the form shape (edit pre-fill). */
// Android configuration arrives as a parsed object (axios); iOS/iPadOS as an
// XML plist string. The Editor needs a string either way.
const configurationToEditorString = (
  configuration: IAppStoreAppVersion["configuration"]
): string => {
  if (!configuration) return "";
  if (typeof configuration === "string") return configuration;
  return JSON.stringify(configuration, null, "\t");
};

export const versionToFormData = (
  version: IAppStoreAppVersion
): IVersionFormData => {
  const { customTarget, targetType, labelTargets } = resolveTargetFromVersion(
    version
  );
  return {
    name: version.name,
    configuration: configurationToEditorString(version.configuration),
    selfService: version.self_service,
    categories: version.categories ?? [],
    targetType,
    customTarget,
    labelTargets,
    autoUpdateEnabled: !!version.auto_update_enabled,
    autoUpdateWindowStart: version.auto_update_window_start ?? "",
    autoUpdateWindowEnd: version.auto_update_window_end ?? "",
  };
};

interface IVersionFormFieldsProps {
  formData: IVersionFormData;
  /** From `useFormValidation`. Text inputs (Name, Configuration, times). */
  setField: <K extends keyof IVersionFormData & string>(
    name: K,
    value: IVersionFormData[K]
  ) => void;
  /** From `useFormValidation`. Compound / toggle-complete controls
   * (Checkbox, Slider, category multi-select, Target radio/dropdown, label
   * selector merges). */
  commitFields: (changes: Partial<IVersionFormData>) => void;
  getError: (name: string) => string | undefined;
  clearFieldError: (name: string) => void;
  validateField: (name: string) => void;
  /** Platform of the title's App Store app. Drives configuration mode
   * (XML for iOS/iPadOS, JSON for Android) and gates the auto-update
   * and self-service sections (iOS/iPadOS only). */
  platform: "ios" | "ipados" | "android" | string;
  /** Display name of the app, used in the auto-update help text. */
  appDisplayName: string;
  labels: ILabelSummary[];
  /** Fleet id. Passed to the shared `CategoriesSelector` so it can load the
   * fleet's self-service categories dynamically (undefined falls back to the
   * hardcoded list). */
  teamId: number;
  /** Click handler for the Category section's "Preview end user experience"
   * button (iOS/iPadOS when self-service is on). Opens the title-level
   * self-service preview modal. */
  onClickPreviewEndUserExperience: () => void;
}

const VersionFormFields = ({
  formData,
  setField,
  commitFields,
  getError,
  clearFieldError,
  validateField,
  platform,
  appDisplayName,
  labels,
  teamId,
  onClickPreviewEndUserExperience,
}: IVersionFormFieldsProps) => {
  const isIosOrIpados = platform === "ios" || platform === "ipados";
  const editorMode = isIosOrIpados ? "xml" : "json";

  const onSelectCategory = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    const category = name as SoftwareCategory;
    const next = value
      ? [...formData.categories, category]
      : formData.categories.filter((c) => c !== category);
    commitFields({ categories: next });
  };

  const onSelectTargetType = (next: string) => {
    commitFields({ targetType: next as VersionTargetType });
  };
  const onSelectCustomTargetOption = (next: string) => {
    commitFields({ customTarget: next as VersionCustomTarget });
  };
  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    commitFields({
      labelTargets: { ...formData.labelTargets, [name]: value },
    });
  };

  const configHelpText = (
    <>
      Updating the configuration will update the app on all hosts in scope.{" "}
      <CustomLink
        newTab
        text="Learn more"
        url={`${LEARN_MORE_ABOUT_BASE_LINK}/${
          isIosOrIpados
            ? "ios-software-managed-configuration"
            : "android-software-managed-configuration"
        }`}
      />
    </>
  );

  return (
    <div className={baseClass}>
      <InputField
        label="Name"
        name="name"
        value={formData.name}
        onChange={(next: string) => setField("name", next)}
        onFocus={() => clearFieldError("name")}
        onBlur={() => validateField("name")}
        error={getError("name")}
        placeholder="e.g. Production"
        autofocus
        inputOptions={{ maxLength: 255 }}
      />

      <Editor
        label="Configuration"
        mode={editorMode}
        value={formData.configuration}
        onChange={(next: string) => setField("configuration", next)}
        helpText={configHelpText}
      />

      {isIosOrIpados && (
        <Slider
          value={formData.selfService}
          onChange={() => commitFields({ selfService: !formData.selfService })}
          activeText="Self-service"
          inactiveText="Self-service"
        />
      )}

      {isIosOrIpados && formData.selfService && (
        <CategoriesSelector
          onSelectCategory={onSelectCategory}
          selectedCategories={formData.categories}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          teamId={teamId}
          label="Category"
          previewButtonLabel="View end user experience"
        />
      )}

      {isIosOrIpados && (
        <FormField name="auto-updates" label="Auto updates">
          <div className={`${baseClass}__auto-updates`}>
            <p className={`${baseClass}__auto-updates-help`}>
              Automatically update <strong>{appDisplayName}</strong> on all
              targeted hosts when a new version is available.
            </p>
            <Checkbox
              value={formData.autoUpdateEnabled}
              onChange={(next: boolean) =>
                commitFields({ autoUpdateEnabled: next })
              }
              name="auto-update-enabled"
            >
              Enable auto updates
            </Checkbox>
            {formData.autoUpdateEnabled && (
              <div className={`${baseClass}__auto-update-window`}>
                <InputField
                  label="Window start (host local time)"
                  name="auto-update-window-start"
                  type="time"
                  value={formData.autoUpdateWindowStart}
                  onChange={(next: string) =>
                    setField("autoUpdateWindowStart", next)
                  }
                  onFocus={() => clearFieldError("autoUpdateWindowStart")}
                  onBlur={() => validateField("autoUpdateWindowStart")}
                  error={getError("autoUpdateWindowStart")}
                  placeholder="00:00"
                />
                <InputField
                  label="Window end (host local time)"
                  name="auto-update-window-end"
                  type="time"
                  value={formData.autoUpdateWindowEnd}
                  onChange={(next: string) =>
                    setField("autoUpdateWindowEnd", next)
                  }
                  onFocus={() => clearFieldError("autoUpdateWindowEnd")}
                  onBlur={() => validateField("autoUpdateWindowEnd")}
                  error={getError("autoUpdateWindowEnd")}
                  placeholder="04:00"
                />
              </div>
            )}
          </div>
        </FormField>
      )}

      <DropdownTargetLabelSelector
        selectedTargetType={formData.targetType}
        selectedCustomTarget={formData.customTarget}
        selectedLabels={formData.labelTargets}
        customTargetOptions={CUSTOM_TARGET_OPTIONS}
        onSelectTargetType={onSelectTargetType}
        onSelectCustomTarget={onSelectCustomTargetOption}
        onSelectLabel={onSelectLabel}
        labels={labels}
        dropdownHelpText={generateHelpText(false, formData.customTarget)}
        error={getError("labelTargets")}
        labelInfoBanner={
          <InfoBanner icon="info-outline">
            If multiple versions target the same host, Fleet will deploy the one
            that was added first.
          </InfoBanner>
        }
      />
    </div>
  );
};

export default VersionFormFields;
