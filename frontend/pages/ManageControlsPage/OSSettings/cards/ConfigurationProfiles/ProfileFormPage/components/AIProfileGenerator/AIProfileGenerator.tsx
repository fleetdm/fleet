import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import FormField from "components/forms/FormField";
import Icon from "components/Icon";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import { IMdmConfig } from "interfaces/config";
import PATHS from "router/paths";
import { AutofillConfigProfilePlatform } from "services/entities/autofill";

import EditTargetsModal, { IProfileTargets } from "../EditTargetsModal";
import GeneratedProfileSummary from "../GeneratedProfileSummary";
import ProfileBottomBar from "../ProfileBottomBar";
import ProfileForm from "../ProfileForm";

import useProfileGeneration, {
  GENERATION_STEP_LABELS,
  IGenerationError,
} from "./useProfileGeneration";

const baseClass = "ai-profile-generator";

// The API's limit, counted in characters as it counts them.
export const MAX_INSTRUCTIONS_LENGTH = 2000;

const OS_SETTINGS_LEARN_MORE_URL = "https://fleetdm.com/os-settings";

// The arrow overlaps the default offset, leaving no gap above the button.
const ACTION_TIP_OFFSET = 10;

const DEFAULT_TARGETS: IProfileTargets = {
  targetType: "All hosts",
  includeMode: "any",
  includeLabels: {},
  excludeLabels: {},
};

const PLATFORM_OPTIONS: {
  label: string;
  value: AutofillConfigProfilePlatform;
}[] = [
  { label: "Apple (macOS, iOS, iPadOS)", value: "apple" },
  { label: "Windows", value: "windows" },
];

const isPlatformMDMOn = (
  platform: AutofillConfigProfilePlatform,
  mdm?: IMdmConfig
) =>
  platform === "apple"
    ? !!mdm?.enabled_and_configured
    : !!mdm?.windows_enabled_and_configured;

/** The platforms the AI can generate for with their MDM turned on. */
export const getAIPlatformOptions = (mdm?: IMdmConfig) =>
  PLATFORM_OPTIONS.filter(({ value }) => isPlatformMDMOn(value, mdm));

const countCharacters = (text: string) => Array.from(text).length;

const getSendTooltip = (instructions: string) => {
  if (!instructions.trim()) {
    return "Write instructions to generate a profile";
  }
  if (countCharacters(instructions) > MAX_INSTRUCTIONS_LENGTH) {
    return `You have exceeded the ${MAX_INSTRUCTIONS_LENGTH} character limit.`;
  }
  return null;
};

const ErrorBanner = ({ error }: { error: IGenerationError }) => (
  <div className={`${baseClass}__error`} role="alert">
    <Icon name="error-outline" color="ui-error" />
    {error.kind === "decline" ? (
      <div className={`${baseClass}__error-message`}>
        Your request could not be processed. {error.reason}{" "}
        <CustomLink
          url={OS_SETTINGS_LEARN_MORE_URL}
          text="Learn more about OS settings"
          newTab
        />
      </div>
    ) : (
      <div className={`${baseClass}__error-message`}>
        Couldn&apos;t generate configuration profile.
        {error.reason ? ` ${error.reason.replace(/\.$/, "")}.` : ""} Please try
        again, or create your profile manually.
      </div>
    )}
  </div>
);

interface IPromptBoxProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  action: React.ReactNode;
}

const PromptBox = ({ value, onChange, readOnly, action }: IPromptBoxProps) => (
  <div className={`${baseClass}__prompt`}>
    <textarea
      aria-label="Instructions"
      className={`${baseClass}__prompt-input`}
      placeholder="Eg., disable the camera on Firefox."
      value={value}
      readOnly={readOnly}
      onChange={(e) => onChange?.(e.target.value)}
      rows={3}
    />
    <div className={`${baseClass}__prompt-action`}>{action}</div>
  </div>
);

interface IAIProfileGeneratorProps {
  router: InjectedRouter;
  teamId: number;
  teamName?: string;
  isPremiumTier: boolean;
  gitOpsModeEnabled: boolean;
  listPath: string;
  /** Where Create manually and the timeout go. */
  manualLocation: { pathname: string; query: Record<string, unknown> };
}

/** The default Add profile view: the admin describes a profile and reviews
 * what the AI generates before adding it. */
const AIProfileGenerator = ({
  router,
  teamId,
  teamName,
  isPremiumTier,
  gitOpsModeEnabled,
  listPath,
  manualLocation,
}: IAIProfileGeneratorProps) => {
  const { config } = useContext(AppContext);
  const platformOptions = getAIPlatformOptions(config?.mdm);
  const [platform, setPlatform] = useState(platformOptions[0]?.value);
  const [instructions, setInstructions] = useState("");
  // Edits to the title and summary made before the profile arrives.
  const [edits, setEdits] = useState<{ name?: string; description?: string }>(
    {}
  );
  // Editable while generating, and kept across retries.
  const [targets, setTargets] = useState<IProfileTargets>(DEFAULT_TARGETS);
  const [showEditTargetsModal, setShowEditTargetsModal] = useState(false);

  const goToManual = (aiTimedOut = false) =>
    router.push({ ...manualLocation, state: { aiTimedOut } });

  const generation = useProfileGeneration({
    onTimeout: () => goToManual(true),
  });
  const { status, nameDescription, profile, error, step, runId } = generation;

  const sendTooltip = getSendTooltip(instructions);
  const canSend = !sendTooltip && !!platform;

  const onGenerate = () => {
    if (!canSend || !platform) {
      return;
    }
    setEdits({});
    generation.generate({
      instructions: instructions.trim(),
      platform,
      // as the upload does: left out for Unassigned and on Free
      fleetId: isPremiumTier && teamId > 0 ? teamId : undefined,
    });
  };

  const renderCreateManually = () => (
    <Button
      variant="subdued"
      icon="arrow"
      iconPosition="right"
      className={`${baseClass}__create-manually`}
      onClick={() => goToManual()}
    >
      Create manually
    </Button>
  );

  const renderSendButton = (label: string, onClick: () => void) => (
    <TooltipWrapper
      tipContent={sendTooltip ?? label}
      position="top"
      showArrow
      tipOffset={ACTION_TIP_OFFSET}
      tooltipClass={`${baseClass}__action-tip`}
      underline={false}
    >
      <Button
        icon="arrow"
        ariaLabel={label}
        className={`${baseClass}__send ${baseClass}__action-button`}
        disabled={!canSend}
        onClick={onClick}
      />
    </TooltipWrapper>
  );

  const renderRetryButton = (isSubmitting: boolean) => (
    <TooltipWrapper
      tipContent={sendTooltip ?? "Retry"}
      position="top"
      showArrow
      tipOffset={ACTION_TIP_OFFSET}
      tooltipClass={`${baseClass}__action-tip`}
      underline={false}
    >
      <Button
        variant="secondary"
        icon="refresh"
        ariaLabel="Retry"
        className={`${baseClass}__action-button`}
        // a save in flight would redirect away from the new generation
        disabled={!canSend || isSubmitting}
        onClick={onGenerate}
      />
    </TooltipWrapper>
  );

  const renderView = () => {
    // The profile and its review form.
    if (status === "done" && profile) {
      const nameFromAI = profile.name ?? nameDescription?.name ?? "";
      return (
        <ProfileForm
          // a retry starts the review over, closing and resetting its options
          key={runId}
          router={router}
          teamId={teamId}
          teamName={teamName}
          isPremiumTier={isPremiumTier}
          gitOpsModeEnabled={gitOpsModeEnabled}
          listPath={listPath}
          generated={{
            type: profile.type,
            contents: profile.content,
            name: edits.name ?? nameFromAI,
            description:
              edits.description ?? nameDescription?.description ?? "",
            isNamePending:
              edits.name === undefined &&
              !profile.name &&
              nameDescription === undefined,
            isDescriptionPending:
              edits.description === undefined && nameDescription === undefined,
            targets,
          }}
          onTargetsChange={setTargets}
          renderBelowEditor={({ isSubmitting }) => (
            <PromptBox
              value={instructions}
              onChange={setInstructions}
              action={renderRetryButton(isSubmitting)}
            />
          )}
        />
      );
    }

    // In flight: placeholders, the progress step and Stop.
    if (status === "generating") {
      return (
        <div className={baseClass}>
          <GeneratedProfileSummary
            name={edits.name ?? nameDescription?.name ?? ""}
            description={
              edits.description ?? nameDescription?.description ?? ""
            }
            isNamePending={
              edits.name === undefined && nameDescription === undefined
            }
            isDescriptionPending={
              edits.description === undefined && nameDescription === undefined
            }
            onChangeName={(name) => setEdits((prev) => ({ ...prev, name }))}
            onChangeDescription={(description) =>
              setEdits((prev) => ({ ...prev, description }))
            }
          />
          <div
            className={`${baseClass}__editor-placeholder`}
            data-testid="editor-placeholder"
            aria-hidden
          />
          <div className={`${baseClass}__progress`} aria-live="polite">
            <Spinner size="x-small" centered={false} delay={0} />
            <span>{GENERATION_STEP_LABELS[step]}</span>
          </div>
          <PromptBox
            value={instructions}
            readOnly
            action={
              <TooltipWrapper
                tipContent="Stop"
                position="top"
                showArrow
                tipOffset={ACTION_TIP_OFFSET}
                tooltipClass={`${baseClass}__action-tip`}
                underline={false}
              >
                <Button
                  variant="alert"
                  ariaLabel="Stop"
                  className={`${baseClass}__stop ${baseClass}__action-button`}
                  onClick={generation.stop}
                >
                  <span className={`${baseClass}__stop-square`} />
                </Button>
              </TooltipWrapper>
            }
          />
          <ProfileBottomBar
            targetType={targets.targetType}
            fleetName={teamName}
            onEditTargets={
              isPremiumTier ? () => setShowEditTargetsModal(true) : undefined
            }
            submitButton={<Button disabled>Add profile</Button>}
          />
        </div>
      );
    }

    // The instructions, with the last error if there was one.
    return (
      <div className={baseClass}>
        <h1>Add profile</h1>
        {isPremiumTier && teamName && (
          <p className={`${baseClass}__fleet`}>
            Adding profile for <b>{teamName}</b>.
          </p>
        )}
        <DropdownWrapper
          label="Platform"
          name="platform"
          className={`${baseClass}__platform`}
          options={platformOptions}
          value={platform}
          onChange={(option) =>
            setPlatform(option?.value as AutofillConfigProfilePlatform)
          }
        />
        <FormField label="Instructions" name="instructions">
          <PromptBox
            value={instructions}
            onChange={setInstructions}
            action={renderSendButton("Generate profile", onGenerate)}
          />
        </FormField>
        {status === "error" && error && <ErrorBanner error={error} />}
        {renderCreateManually()}
      </div>
    );
  };

  return (
    <>
      {renderView()}
      {/* Outside the views, so a profile arriving mid-edit doesn't close it;
          the review form picks up what's chosen here. */}
      {showEditTargetsModal && (
        <EditTargetsModal
          fleetId={teamId}
          targets={targets}
          onAddLabel={() => router.push(PATHS.LABEL_NEW_DYNAMIC)}
          onDone={(newTargets) => {
            setTargets(newTargets);
            setShowEditTargetsModal(false);
          }}
          onExit={() => setShowEditTargetsModal(false)}
        />
      )}
    </>
  );
};

export default AIProfileGenerator;
