import { AxiosResponse } from "axios";
import React, { useContext, useEffect, useRef, useState } from "react";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Editor from "components/Editor";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import useBlockNavigation from "hooks/useBlockNavigation";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { IApiError } from "interfaces/errors";
import {
  IMdmProfile,
  IProfileLabel,
  isAnyMDMConfigured,
  isMDMConfiguredForPlatform,
  platformToMDMLabel,
} from "interfaces/mdm";
import PATHS from "router/paths";
import mdmAPI from "services/entities/mdm";

import {
  ADD_PROFILE_ACCEPT,
  contentTypeForExtension,
  detectProfileContentType,
  generateCustomTargetLabelKey,
  getAcceptedExtensions,
  getErrorMessage,
  isProfileNameTakenError,
  PASTED_PROFILE_DEFAULT_NAME,
  nextPastedProfileName,
  parseFile,
  PROFILE_CONTENT_TYPES,
  ProfileContentType,
  profileContentTypeFor,
} from "../../../components/ProfileUploader/helpers";
import EditTargetsModal, { IProfileTargets } from "../EditTargetsModal";
import GeneratedProfileSummary from "../GeneratedProfileSummary";
import ProfileAdvancedOptions, {
  IProfileDeployOptions,
} from "../ProfileAdvancedOptions";
import ProfileBottomBar from "../ProfileBottomBar";

const baseClass = "profile-form";

// Match the name and description columns (server/datastore/mysql/schema.sql).
const NAME_MAX_LENGTH = 255;
const DESCRIPTION_MAX_LENGTH = 1023;

// The editor stays short until there is something to show, per the design.
const EMPTY_EDITOR_MAX_LINES = 3;
const EDITOR_MAX_LINES = 30;

// Same page size as the policy automations' profile list; the endpoint's
// default isn't documented.
const PROFILE_NAMES_PER_PAGE = 1000;

const UNRECOGNIZED_CONTENTS_ERROR =
  "Paste a .mobileconfig, declaration (.json), Android (.json) or Windows (.xml) profile";

const labelsToSelection = (labels?: IProfileLabel[]) =>
  (labels ?? []).reduce<Record<string, boolean>>((selection, label) => {
    selection[label.name] = true;
    return selection;
  }, {});

interface IProfileFormData extends IProfileTargets, IProfileDeployOptions {
  name: string;
  description: string;
  contents: string;
}

interface IEditedProfile {
  profile: IMdmProfile;
  contents: string;
}

/** A profile the AI generated, reviewed here before it's added. */
export interface IGeneratedProfile {
  /** From the API's response, so the contents aren't typed again. */
  type: ProfileContentType;
  contents: string;
  /** Change after mount when the name and description arrive late; a field
   * the admin has edited keeps the edit. */
  name: string;
  description: string;
  isNamePending?: boolean;
  isDescriptionPending?: boolean;
}

/** An existing profile's values, a generated profile's, or an empty form. */
const initialFormDataFor = (
  editing?: IEditedProfile,
  generated?: IGeneratedProfile
): IProfileFormData => {
  const profile = editing?.profile;
  const includeLabels =
    profile?.labels_include_all ?? profile?.labels_include_any;
  const excludeLabels = profile?.labels_exclude_any;
  return {
    name: profile?.name ?? generated?.name ?? "",
    description: profile?.description ?? generated?.description ?? "",
    contents: editing?.contents ?? generated?.contents ?? "",
    targetType:
      includeLabels?.length || excludeLabels?.length ? "Custom" : "All hosts",
    includeMode: profile?.labels_include_all?.length ? "all" : "any",
    includeLabels: labelsToSelection(includeLabels),
    excludeLabels: labelsToSelection(excludeLabels),
    selfService: profile?.self_service ?? false,
    hidden: profile?.hidden ?? false,
  };
};

interface IUploadedFile {
  name: string;
  /** The type its extension implies, used when the contents don't say. */
  type: ProfileContentType | null;
}

interface IProfileFormProps {
  router: InjectedRouter;
  teamId: number;
  teamName?: string;
  isPremiumTier: boolean;
  gitOpsModeEnabled: boolean;
  listPath: string;
  /** The profile being edited and its contents; absent when adding. */
  editing?: IEditedProfile;
  /** Shows the AI's title and summary in place of the name and description
   * fields. Only when adding. */
  generated?: IGeneratedProfile;
  /** Rendered under the editor, given whether the profile is being saved. */
  renderBelowEditor?: (state: { isSubmitting: boolean }) => React.ReactNode;
}

const ProfileForm = ({
  router,
  teamId,
  teamName,
  isPremiumTier,
  gitOpsModeEnabled,
  listPath,
  editing,
  generated,
  renderBelowEditor,
}: IProfileFormProps) => {
  const { config } = useContext(AppContext);
  const profile = editing?.profile;
  const isEdit = !!profile;
  const [initialFormData] = useState(() =>
    initialFormDataFor(editing, generated)
  );
  // The uploaded file only seeds the editor; its name is what the server
  // derives a profile name from when the admin leaves Name empty.
  const [uploadedFile, setUploadedFile] = useState<IUploadedFile | null>(null);
  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);
  const [showEditTargetsModal, setShowEditTargetsModal] = useState(false);
  // Open when editing a profile that isn't on the defaults, so its deploy
  // settings aren't tucked away.
  const [isAdvancedOptionsOpen, setIsAdvancedOptionsOpen] = useState(
    initialFormData.selfService || initialFormData.hidden
  );
  const fileInputRef = useRef<HTMLInputElement>(null);

  // An uploaded file the contents can't type, like a lone secret placeholder,
  // goes by its extension, as the server routes it.
  const detectContentType = (contents: string) =>
    detectProfileContentType(contents) ?? uploadedFile?.type ?? null;

  const validate = (data: IProfileFormData): IFormErrors => {
    const errors: IFormErrors = {};
    // On add an empty name means "derive it from the file"; an existing
    // profile has nothing to derive from, so it can't be blanked.
    if (isEdit && !data.name.trim()) {
      errors.name = "Enter a name";
    }
    if (!data.contents.trim()) {
      errors.contents = "Upload or paste a profile";
    } else if (
      !isEdit &&
      !(generated?.type ?? detectContentType(data.contents))
    ) {
      errors.contents = UNRECOGNIZED_CONTENTS_ERROR;
    }
    return errors;
  };

  const {
    formData,
    setField,
    commitFields,
    getTextFieldProps,
    clearFieldError,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IProfileFormData>({
    initialFormData,
    validate,
    serverErrors,
    // sent byte for byte, so a pasted profile isn't altered
    skipTrim: ["contents"],
  });

  // A late name or description replaces only what the admin hasn't edited.
  const appliedGeneratedRef = useRef({
    name: initialFormData.name,
    description: initialFormData.description,
  });
  const formDataRef = useRef(formData);
  formDataRef.current = formData;
  const generatedName = generated?.name;
  const generatedDescription = generated?.description;
  useEffect(() => {
    const applied = appliedGeneratedRef.current;
    const current = formDataRef.current;
    if (generatedName !== undefined && generatedName !== applied.name) {
      if (current.name === applied.name) {
        setField("name", generatedName);
      }
      applied.name = generatedName;
    }
    if (
      generatedDescription !== undefined &&
      generatedDescription !== applied.description
    ) {
      if (current.description === applied.description) {
        setField("description", generatedDescription);
      }
      applied.description = generatedDescription;
    }
  }, [generatedName, generatedDescription, setField]);

  // An existing profile's type is fixed by what it is, and a generated one's
  // by the API; pasted content on the add page is typed by what it looks like.
  const contentType: ProfileContentType | null = profile
    ? profileContentTypeFor(profile)
    : generated?.type ?? detectContentType(formData.contents);
  const hasContents = formData.contents.trim() !== "";

  // Same gate as the profiles list; this page is also reachable by URL and
  // from the command palette.
  const isAnyMDMEnabled = isAnyMDMConfigured(config?.mdm);
  const platform =
    profile?.platform ??
    (contentType ? PROFILE_CONTENT_TYPES[contentType].platform : undefined);
  const isPlatformMDMEnabled = platform
    ? isMDMConfiguredForPlatform(platform, config?.mdm)
    : isAnyMDMEnabled;
  // Fields stay editable when only the pasted type's MDM is off, so the admin
  // can replace the contents; submitting is what gets blocked.
  const isMDMEnabled = profile ? isPlatformMDMEnabled : isAnyMDMEnabled;

  // Only a .mobileconfig can be self-service; replacing the contents with
  // another type resets it to force.
  const canSelfService = contentType === "mobileconfig";
  const { selfService, hidden } = formData;

  const labelKey = generateCustomTargetLabelKey(formData);
  const initialLabelKey = generateCustomTargetLabelKey(initialFormData);

  // compared trimmed, as the server stores them, so a stored value with
  // surrounding spaces doesn't read as an edit
  const nameChanged = formData.name.trim() !== initialFormData.name.trim();
  const descriptionChanged =
    formData.description.trim() !== initialFormData.description.trim();
  const contentsChanged = formData.contents !== initialFormData.contents;
  const targetChanged =
    JSON.stringify(labelKey) !== JSON.stringify(initialLabelKey);
  const deployChanged =
    selfService !== initialFormData.selfService ||
    hidden !== initialFormData.hidden;
  const hasChanges =
    nameChanged ||
    descriptionChanged ||
    contentsChanged ||
    targetChanged ||
    deployChanged;
  // Free can't set either flag, so it sends neither.
  const deployFields = isPremiumTier ? { selfService, hidden } : {};
  // A generated profile is lost on leaving, edited or not.
  useBlockNavigation(hasChanges || !!generated || isSubmitting);

  // An existing or generated profile's type is fixed; on add the new
  // contents decide it.
  const keepsSelfService = (type: ProfileContentType | null) =>
    !!profile || (generated?.type ?? type) === "mobileconfig";

  const onFileSelected = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    const file = files[0];
    let details;
    try {
      details = await parseFile(file);
      if (
        profile &&
        !getAcceptedExtensions(profile).includes(`.${details.ext}`)
      ) {
        throw new Error(`Invalid file type: ${details.ext}`);
      }
    } catch (e) {
      notify.error("Invalid file type", { response: e });
      return;
    }
    try {
      const contents = await file.text();
      const type = contentTypeForExtension(details.ext);
      commitFields({
        contents,
        ...(!keepsSelfService(detectProfileContentType(contents) ?? type) && {
          selfService: false,
        }),
      });
      setUploadedFile({ name: details.name, type });
      // commitFields validated without the file's type, so a stale error can
      // survive it; submit checks the new contents again.
      clearFieldError("contents");
    } catch (e) {
      notify.error("Couldn't read the file. Please try again.", {
        response: e,
      });
    }
  };

  const onFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onFileSelected(e.target.files);
    // let the same file be picked again after a paste
    e.target.value = "";
  };

  // Typed or pasted contents are no longer the uploaded file, so the derived
  // name and type shouldn't come from it.
  const onContentsChange = (value: string) => {
    setField("contents", value);
    if (!keepsSelfService(detectProfileContentType(value))) {
      setField("selfService", false);
    }
    setUploadedFile(null);
  };

  const buildFile = (
    contents: string,
    pastedName = PASTED_PROFILE_DEFAULT_NAME
  ) => {
    const ext = contentType ? PROFILE_CONTENT_TYPES[contentType].extension : "";
    const fileName = uploadedFile?.name ?? pastedName;
    return new File([contents], `${fileName}.${ext}`, { type: "text/plain" });
  };

  // The server names unnamed .xml and .json profiles after the file, so a
  // pasted one gets the next free default. On failure the server's
  // duplicate-name error still applies.
  const getPastedName = async (name: string) => {
    if (name || uploadedFile || contentType === "mobileconfig") {
      return undefined;
    }
    try {
      const { profiles } = await mdmAPI.getProfiles({
        fleet_id: teamId,
        page: 0,
        per_page: PROFILE_NAMES_PER_PAGE,
      });
      return nextPastedProfileName((profiles ?? []).map((p) => p.name));
    } catch {
      return undefined;
    }
  };

  // Receives the form data trimmed, except the contents.
  const onValidSubmit = async (data: IProfileFormData) => {
    if (gitOpsModeEnabled) {
      return;
    }
    // A PATCH with nothing in it still logs an edit, and for Android marks
    // the profile pending on hosts.
    if (profile && !hasChanges) {
      router.push(listPath);
      return;
    }
    try {
      if (!profile) {
        await mdmAPI.uploadProfile({
          file: buildFile(data.contents, await getPastedName(data.name)),
          teamId,
          name: data.name || undefined,
          description: data.description || undefined,
          ...labelKey,
          ...deployFields,
        });
        notify.success("Successfully uploaded.");
      } else {
        // labels use replace semantics on the API, so always submit the full
        // current label selection even when only the contents changed.
        await mdmAPI.updateProfile({
          profileUUID: profile.profile_uuid,
          profile: contentsChanged ? buildFile(data.contents) : undefined,
          name: nameChanged ? data.name : undefined,
          description: descriptionChanged ? data.description : undefined,
          ...labelKey,
          ...deployFields,
        });
        notify.success("Successfully updated profile.");
      }
      router.push(listPath);
    } catch (e) {
      const err = e as AxiosResponse<IApiError>;
      const message = getErrorMessage(err, isEdit ? "edit" : "add");
      if (isProfileNameTakenError(err) && typeof message === "string") {
        // shown on the name field, and toasted by the form
        setServerErrors({ name: message });
      } else {
        notify.error(message, { response: e });
      }
    }
  };

  const onAddLabel = () => router.push(PATHS.LABEL_NEW_DYNAMIC);

  const onDoneEditingTargets = (targets: IProfileTargets) => {
    commitFields(targets);
    setShowEditTargetsModal(false);
  };

  const disabled = gitOpsModeEnabled || !isMDMEnabled;
  const isFieldDisabled = (disableChildren?: boolean) =>
    !!disableChildren || !isMDMEnabled || isSubmitting;

  // On add there's nothing to deploy until there's a profile, per the design.
  const showAdvancedOptions = isPremiumTier && (isEdit || hasContents);

  const renderSubmitButton = () => {
    const btn = (
      <Button
        type="submit"
        className={`${baseClass}__submit`}
        isLoading={isSubmitting}
        // Invalid values are reported on submit. Only the empty add page stays
        // disabled, per the design.
        disabled={
          disabled ||
          !isPlatformMDMEnabled ||
          isSubmitting ||
          (!isEdit && !hasContents) ||
          // a save before they arrive would drop the generated name and summary
          !!generated?.isNamePending ||
          !!generated?.isDescriptionPending
        }
      >
        {isEdit ? "Update profile" : "Add profile"}
      </Button>
    );

    if (!isPlatformMDMEnabled) {
      return (
        <TooltipWrapper
          tipContent={
            <p>
              To enable, first turn on{" "}
              <CustomLink
                text={platform ? `${platformToMDMLabel(platform)} MDM` : "MDM"}
                url={PATHS.ADMIN_INTEGRATIONS_MDM}
                variant="tooltip-link"
              />
              .
            </p>
          }
          showArrow
          position="top"
          underline={false}
        >
          {btn}
        </TooltipWrapper>
      );
    }
    return (
      <GitOpsModeTooltipWrapper position="top" renderChildren={() => btn} />
    );
  };

  return (
    <>
      <form className={`${baseClass}`} onSubmit={handleSubmit(onValidSubmit)}>
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(disableChildren) => (
            <div className={`${baseClass}__fields`}>
              {generated ? (
                <GeneratedProfileSummary
                  name={formData.name}
                  description={formData.description}
                  isNamePending={generated.isNamePending}
                  isDescriptionPending={generated.isDescriptionPending}
                  disabled={isFieldDisabled(disableChildren)}
                  onChangeName={(name) => commitFields({ name })}
                  onChangeDescription={(description) =>
                    commitFields({ description })
                  }
                />
              ) : (
                <>
                  <InputField
                    {...getTextFieldProps("name")}
                    label="Name"
                    placeholder="E.g., Disable camera access for Firefox"
                    inputOptions={{ maxLength: NAME_MAX_LENGTH }}
                    disabled={isFieldDisabled(disableChildren)}
                  />
                  <InputField
                    {...getTextFieldProps("description")}
                    label="Description"
                    type="textarea"
                    placeholder="E.g., Blocks Firefox from accessing the camera on managed Macs, without affecting other apps."
                    inputOptions={{ maxLength: DESCRIPTION_MAX_LENGTH }}
                    disabled={isFieldDisabled(disableChildren)}
                  />
                  <div className={`${baseClass}__upload`}>
                    <Button
                      variant="secondary"
                      icon="upload"
                      disabled={isFieldDisabled(disableChildren)}
                      onClick={() => fileInputRef.current?.click()}
                    >
                      Upload a profile
                    </Button>
                    <input
                      ref={fileInputRef}
                      accept={
                        profile
                          ? getAcceptedExtensions(profile).join(",")
                          : ADD_PROFILE_ACCEPT
                      }
                      id="upload-profile"
                      aria-label="Upload a profile"
                      type="file"
                      disabled={isFieldDisabled(disableChildren)}
                      onChange={onFileInputChange}
                    />
                  </div>
                </>
              )}
              <Editor
                {...getTextFieldProps("contents")}
                name="profile-contents"
                label={
                  contentType
                    ? PROFILE_CONTENT_TYPES[contentType].label
                    : undefined
                }
                ariaLabel="Profile contents"
                mode={
                  contentType
                    ? PROFILE_CONTENT_TYPES[contentType].editorMode
                    : "text"
                }
                onChange={onContentsChange}
                placeholder="// Paste or type your profile into here //"
                minLines={EMPTY_EDITOR_MAX_LINES}
                maxLines={
                  hasContents ? EDITOR_MAX_LINES : EMPTY_EDITOR_MAX_LINES
                }
                showPrintMargin={false}
                // Ace's XML behaviours auto-insert closing tags while typing,
                // which corrupts a profile typed in by hand.
                onLoad={(editor) => editor.setBehavioursEnabled(false)}
                readOnly={isFieldDisabled(disableChildren)}
                wrapEnabled
                helpText={
                  generated
                    ? "AI can make mistakes. Please double-check before you proceed."
                    : undefined
                }
              />
              {renderBelowEditor?.({ isSubmitting })}
            </div>
          )}
        />
        {showAdvancedOptions && (
          <ProfileAdvancedOptions
            isOpen={isAdvancedOptionsOpen}
            onToggle={() => setIsAdvancedOptionsOpen(!isAdvancedOptionsOpen)}
            canSelfService={canSelfService}
            selfService={selfService}
            hidden={hidden}
            onChange={commitFields}
            disabled={isFieldDisabled()}
          />
        )}
        <ProfileBottomBar
          targetType={formData.targetType}
          fleetName={teamName}
          onEditTargets={
            isPremiumTier ? () => setShowEditTargetsModal(true) : undefined
          }
          editTargetsDisabled={isSubmitting}
          submitButton={renderSubmitButton()}
        />
      </form>
      {/* Outside the form, so Enter in the label search doesn't submit the
          profile. */}
      {showEditTargetsModal && (
        <EditTargetsModal
          fleetId={teamId}
          targets={{
            targetType: formData.targetType,
            includeMode: formData.includeMode,
            includeLabels: formData.includeLabels,
            excludeLabels: formData.excludeLabels,
          }}
          disabled={!isMDMEnabled}
          onAddLabel={onAddLabel}
          onDone={onDoneEditingTargets}
          onExit={() => setShowEditTargetsModal(false)}
        />
      )}
    </>
  );
};

export default ProfileForm;
