import { AxiosResponse } from "axios";
import React, { useContext, useEffect, useRef, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, RouteComponentProps } from "react-router";

import BackButton from "components/BackButton";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Editor from "components/Editor";
import InputField from "components/forms/fields/InputField";
import FormField from "components/forms/FormField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import {
  TargetLabelSelector,
  ILabelConfig,
  LabelTargetMode,
  TargetType,
} from "components/TargetLabelSelector";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import useBlockNavigation from "hooks/useBlockNavigation";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import useGitOpsMode from "hooks/useGitOpsMode";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import {
  IMdmProfile,
  IProfileLabel,
  isAnyMDMConfigured,
  isMDMConfiguredForPlatform,
  platformToMDMLabel,
} from "interfaces/mdm";
import { API_NO_TEAM_ID } from "interfaces/team";
import PATHS from "router/paths";
import configProfileAPI from "services/entities/config_profiles";
import labelsAPI, {
  getCustomLabels,
  listNamesFromSelectedLabels,
} from "services/entities/labels";
import mdmAPI from "services/entities/mdm";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import {
  ADD_PROFILE_ACCEPT,
  detectProfileContentType,
  editorModeForContentType,
  generateCustomTargetLabelKey,
  getAcceptedExtensions,
  getErrorMessage,
  isProfileNameTakenError,
  PASTED_PROFILE_DEFAULT_NAME,
  nextPastedProfileName,
  parseFile,
  PROFILE_CONTENT_TYPE_EXTENSION,
  PROFILE_CONTENT_TYPE_LABEL,
  PROFILE_CONTENT_TYPE_PLATFORM,
  ProfileContentType,
  profileContentTypeFor,
} from "../components/ProfileUploader/helpers";

const baseClass = "profile-form-page";

// Match the name and description columns (server/datastore/mysql/schema.sql).
const NAME_MAX_LENGTH = 255;
const DESCRIPTION_MAX_LENGTH = 1023;

// The editor stays short until there is something to show, per the design.
const EMPTY_EDITOR_MAX_LINES = 3;
const EDITOR_MAX_LINES = 30;

const UNRECOGNIZED_CONTENTS_ERROR =
  "Paste a .mobileconfig, declaration (.json), Android (.json) or Windows (.xml) profile";

const labelsToSelection = (labels?: IProfileLabel[]) =>
  (labels ?? []).reduce<Record<string, boolean>>((selection, label) => {
    selection[label.name] = true;
    return selection;
  }, {});

interface IProfileFormData {
  name: string;
  description: string;
  contents: string;
  targetType: TargetType;
  includeMode: LabelTargetMode;
  includeLabels: Record<string, boolean>;
  excludeLabels: Record<string, boolean>;
}

interface IEditedProfile {
  profile: IMdmProfile;
  contents: string;
}

/** An existing profile's values, or an empty form when adding. */
const initialFormDataFor = (editing?: IEditedProfile): IProfileFormData => {
  const profile = editing?.profile;
  const includeLabels =
    profile?.labels_include_all ?? profile?.labels_include_any;
  const excludeLabels = profile?.labels_exclude_any;
  return {
    name: profile?.name ?? "",
    description: profile?.description ?? "",
    contents: editing?.contents ?? "",
    targetType:
      includeLabels?.length || excludeLabels?.length ? "Custom" : "All hosts",
    includeMode: profile?.labels_include_all?.length ? "all" : "any",
    includeLabels: labelsToSelection(includeLabels),
    excludeLabels: labelsToSelection(excludeLabels),
  };
};

interface IProfileFormProps {
  router: InjectedRouter;
  teamId: number;
  teamName?: string;
  isPremiumTier: boolean;
  gitOpsModeEnabled: boolean;
  listPath: string;
  /** The profile being edited and its contents; absent when adding. */
  editing?: IEditedProfile;
}

const ProfileForm = ({
  router,
  teamId,
  teamName,
  isPremiumTier,
  gitOpsModeEnabled,
  listPath,
  editing,
}: IProfileFormProps) => {
  const { config } = useContext(AppContext);
  const profile = editing?.profile;
  const isEdit = !!profile;
  const [initialFormData] = useState(() => initialFormDataFor(editing));
  // The uploaded file only seeds the editor; its name is what the server
  // derives a profile name from when the admin leaves Name empty.
  const [uploadedFileName, setUploadedFileName] = useState<string | null>(null);
  const [serverErrors, setServerErrors] = useState<IFormErrors | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const validate = (data: IProfileFormData): IFormErrors => {
    const errors: IFormErrors = {};
    // On add an empty name means "derive it from the file"; an existing
    // profile has nothing to derive from, so it can't be blanked.
    if (isEdit && !data.name.trim()) {
      errors.name = "Enter a name";
    }
    if (!data.contents.trim()) {
      errors.contents = "Upload or paste a profile";
    } else if (!isEdit && !detectProfileContentType(data.contents)) {
      errors.contents = UNRECOGNIZED_CONTENTS_ERROR;
    }
    if (
      data.targetType === "Custom" &&
      !listNamesFromSelectedLabels(data.includeLabels).length &&
      !listNamesFromSelectedLabels(data.excludeLabels).length
    ) {
      errors.target = "Select at least one label";
    }
    return errors;
  };

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IProfileFormData>({
    initialFormData,
    validate,
    serverErrors,
    // sent byte for byte, so a pasted profile isn't altered
    skipTrim: ["contents"],
  });

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isFetching: isFetchingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", teamId],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: isPremiumTier }
  );

  // An existing profile's type is fixed by what it is; pasted content on the
  // add page is typed by what it looks like.
  const contentType: ProfileContentType | null = profile
    ? profileContentTypeFor(profile)
    : detectProfileContentType(formData.contents);
  const hasContents = formData.contents.trim() !== "";

  // Same gate as the profiles list; this page is also reachable by URL and
  // from the command palette.
  const isAnyMDMEnabled = isAnyMDMConfigured(config?.mdm);
  const platform =
    profile?.platform ??
    (contentType ? PROFILE_CONTENT_TYPE_PLATFORM[contentType] : undefined);
  const isPlatformMDMEnabled = platform
    ? isMDMConfiguredForPlatform(platform, config?.mdm)
    : isAnyMDMEnabled;
  // Fields stay editable when only the pasted type's MDM is off, so the admin
  // can replace the contents; submitting is what gets blocked.
  const isMDMEnabled = profile ? isPlatformMDMEnabled : isAnyMDMEnabled;

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
  const hasChanges =
    nameChanged || descriptionChanged || contentsChanged || targetChanged;
  useBlockNavigation(hasChanges && !isSubmitting);

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
      commitFields({ contents: await file.text() });
      setUploadedFileName(details.name);
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
  // name shouldn't come from it.
  const onContentsChange = (value: string) => {
    setField("contents", value);
    setUploadedFileName(null);
  };

  const buildFile = (
    contents: string,
    pastedName = PASTED_PROFILE_DEFAULT_NAME
  ) => {
    const ext = contentType ? PROFILE_CONTENT_TYPE_EXTENSION[contentType] : "";
    const fileName = uploadedFileName ?? pastedName;
    return new File([contents], `${fileName}.${ext}`, { type: "text/plain" });
  };

  // The server names unnamed .xml and .json profiles after the file, so a
  // pasted one gets the next free default. On failure the server's
  // duplicate-name error still applies.
  const getPastedName = async (name: string) => {
    if (name || uploadedFileName || contentType === "mobileconfig") {
      return undefined;
    }
    try {
      const { profiles } = await mdmAPI.getProfiles({ fleet_id: teamId });
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
    try {
      if (!profile) {
        await mdmAPI.uploadProfile({
          file: buildFile(data.contents, await getPastedName(data.name)),
          teamId,
          name: data.name || undefined,
          description: data.description || undefined,
          ...labelKey,
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

  const includeTab: ILabelConfig = {
    selectedLabels: formData.includeLabels,
    onSelectLabel: ({ name: labelName, value }) =>
      commitFields({
        includeLabels: { ...formData.includeLabels, [labelName]: value },
      }),
    showModeToggle: true,
    mode: formData.includeMode,
    onSelectMode: (includeMode) => commitFields({ includeMode }),
    anyTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have any</b>
        </em>{" "}
        of these labels.
      </>
    ),
    allTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have all</b>
        </em>{" "}
        of these labels.
      </>
    ),
  };

  const excludeTab: ILabelConfig = {
    selectedLabels: formData.excludeLabels,
    onSelectLabel: ({ name: labelName, value }) =>
      commitFields({
        excludeLabels: { ...formData.excludeLabels, [labelName]: value },
      }),
  };

  const onAddLabel = () => router.push(PATHS.LABEL_NEW_DYNAMIC);

  const disabled = gitOpsModeEnabled || !isMDMEnabled;
  const isFieldDisabled = (disableChildren?: boolean) =>
    !!disableChildren || !isMDMEnabled || isSubmitting;

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
          (!isEdit && !hasContents)
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
    <form
      className={`${baseClass}__form`}
      onSubmit={handleSubmit(onValidSubmit)}
    >
      <GitOpsModeTooltipWrapper
        isInputField
        renderChildren={(disableChildren) => (
          <div className={`${baseClass}__fields`}>
            <InputField
              label="Name"
              name="name"
              value={formData.name}
              error={getError("name")}
              onChange={(value: string) => setField("name", value)}
              onFocus={() => clearFieldError("name")}
              onBlur={() => validateField("name")}
              placeholder="E.g., Disable camera access for Firefox"
              inputOptions={{ maxLength: NAME_MAX_LENGTH }}
              disabled={isFieldDisabled(disableChildren)}
            />
            <InputField
              label="Description"
              name="description"
              type="textarea"
              value={formData.description}
              onChange={(value: string) => setField("description", value)}
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
              {(hasContents || isEdit) && (
                <span className={`${baseClass}__upload-help`}>
                  Uploading a file will replace the profile contents below.
                </span>
              )}
            </div>
            <Editor
              name="profile-contents"
              label={
                contentType
                  ? PROFILE_CONTENT_TYPE_LABEL[contentType]
                  : undefined
              }
              ariaLabel="Profile contents"
              error={getError("contents")}
              mode={editorModeForContentType(contentType)}
              value={formData.contents}
              onChange={onContentsChange}
              onFocus={() => clearFieldError("contents")}
              onBlur={() => validateField("contents")}
              placeholder="// Or paste your profile into here //"
              minLines={EMPTY_EDITOR_MAX_LINES}
              maxLines={hasContents ? EDITOR_MAX_LINES : EMPTY_EDITOR_MAX_LINES}
              showPrintMargin={false}
              // Ace's XML behaviours auto-insert closing tags while typing,
              // which corrupts a profile typed in by hand.
              onLoad={(editor) => editor.setBehavioursEnabled(false)}
              readOnly={isFieldDisabled(disableChildren)}
              wrapEnabled
            />
          </div>
        )}
      />
      {isPremiumTier && (
        <GitOpsModeTooltipWrapper
          isInputField
          renderChildren={(disableChildren) => (
            <FormField
              label="Target"
              name="target"
              error={getError("target")}
              className={`${baseClass}__target`}
            >
              <TargetLabelSelector
                selectedTargetType={formData.targetType}
                onSelectTargetType={(targetType) =>
                  commitFields({ targetType })
                }
                labels={labels || []}
                includeConfig={includeTab}
                excludeConfig={excludeTab}
                isLoadingLabels={isLoadingLabels || isFetchingLabels}
                isErrorLabels={isErrorLabels}
                emptyStateDescription="Add a label to target your configuration profile."
                onAddLabel={onAddLabel}
                disableOptions={isFieldDisabled(disableChildren)}
              />
            </FormField>
          )}
        />
      )}
      <div className={`${baseClass}__footer`}>
        <span className={`${baseClass}__applies-to`}>
          Applies to{" "}
          <b>
            {formData.targetType === "Custom"
              ? "hosts in custom targets"
              : "all hosts"}
          </b>
          {teamName && (
            <>
              {" "}
              for <b>{teamName}</b>
            </>
          )}
          .
        </span>
        <div className={`${baseClass}__footer-actions`}>
          <Button variant="secondary" onClick={() => router.push(listPath)}>
            Cancel
          </Button>
          {renderSubmitButton()}
        </div>
      </div>
    </form>
  );
};

interface IProfileFormPageRouteParams {
  profile_uuid?: string;
}

type IProfileFormPageProps = RouteComponentProps<
  undefined,
  IProfileFormPageRouteParams
>;

/** Add (no profile_uuid) or edit (profile_uuid) a configuration profile as a
 * full page, with the contents editable inline. */
const ProfileFormPage = ({
  router,
  routeParams,
  location,
}: IProfileFormPageProps) => {
  const profileUUID = routeParams.profile_uuid;
  const {
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();

  const { currentTeamName, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
      observer_plus: false,
      technician: false,
    },
  });
  // Same coercion as ManageControlsPage: free tier only has "No fleet".
  const teamId = isPremiumTier === false ? API_NO_TEAM_ID : teamIdForApi;

  const listPath = getPathWithQueryParams(PATHS.CONTROLS_CUSTOM_SETTINGS, {
    fleet_id: isPremiumTier ? teamId : undefined,
  });

  const {
    data: profile,
    isLoading: isLoadingProfile,
    isError: isErrorProfile,
  } = useQuery<IMdmProfile, Error>(
    ["config_profile", profileUUID],
    () => configProfileAPI.getConfigProfile(profileUUID as string),
    // The form is seeded once from this data, so a cached copy from before
    // the last save would be edited and written back. Always fetch fresh.
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!profileUUID, cacheTime: 0 }
  );
  const {
    data: contents,
    isLoading: isLoadingContents,
    isError: isErrorContents,
  } = useQuery<string, Error>(
    ["config_profile_contents", profileUUID],
    // as text, so JSON profiles are edited byte for byte rather than as a
    // re-serialized copy
    () => mdmAPI.downloadProfile(profileUUID as string, "text"),
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!profileUUID, cacheTime: 0 }
  );

  // The edit URL is shareable, so its fleet_id can disagree with the fleet
  // the profile belongs to; follow the profile so labels and the footer match.
  const profileTeamId = profile ? profile.team_id ?? API_NO_TEAM_ID : undefined;
  const isWrongFleet =
    !!isPremiumTier &&
    profileTeamId !== undefined &&
    teamId !== undefined &&
    profileTeamId !== teamId;
  useEffect(() => {
    if (isWrongFleet) {
      router.replace(
        getPathWithQueryParams(location.pathname, {
          fleet_id: profileTeamId,
        })
      );
    }
  }, [isWrongFleet, location.pathname, profileTeamId, router]);

  // The roles useTeamIdParam allows above; anyone else has no fleet to land
  // on and would otherwise wait on a spinner.
  const canEditProfiles =
    isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin;

  const renderContent = () => {
    if (!canEditProfiles) {
      return (
        <DataError description="You don't have permission to edit profiles." />
      );
    }
    if (teamId === undefined) {
      return <Spinner />;
    }
    if (
      profileUUID &&
      (isLoadingProfile || isLoadingContents || isWrongFleet)
    ) {
      return <Spinner />;
    }
    if (profileUUID && (isErrorProfile || isErrorContents || !profile)) {
      return <DataError description="Couldn't load the profile." />;
    }
    return (
      <ProfileForm
        // remount when switching profiles so the form re-seeds
        key={profileUUID ?? "new"}
        router={router}
        teamId={teamId}
        teamName={currentTeamName}
        isPremiumTier={!!isPremiumTier}
        gitOpsModeEnabled={gitOpsModeEnabled}
        listPath={listPath}
        editing={profile ? { profile, contents: contents ?? "" } : undefined}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <BackButton text="Back to profiles" path={listPath} />
      <h1>{profileUUID ? "Edit profile" : "Add profile"}</h1>
      {renderContent()}
    </MainContent>
  );
};

export default ProfileFormPage;
