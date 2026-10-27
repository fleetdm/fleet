import { AxiosResponse } from "axios";
import { useCallback, useEffect, useRef, useState } from "react";

import { IApiError } from "interfaces/errors";
import autofillAPI, {
  AutofillConfigProfilePlatform,
  IAutofillConfigProfileContentResponse,
  IAutofillConfigProfileNameDescriptionResponse,
} from "services/entities/autofill";

/** How long "Reading request" shows before the next step, per the design. */
export const READING_STEP_MS = 1000;
/** Product's cap on a generation; past it the admin is sent to the manual view. */
export const GENERATION_TIMEOUT_MS = 10000;

// Fleet's own call to the AI service gives up just inside the cap, so its
// timeout is the cap being reached.
const SERVER_TIMEOUT_REASON = "The AI service timed out.";

export type GenerationStep = "reading" | "drafting" | "writing";

export const GENERATION_STEP_LABELS: Record<GenerationStep, string> = {
  reading: "Reading request",
  drafting: "Drafting name and summary",
  writing: "Writing and validating profile",
};

export interface IGenerationError {
  /** "decline" when the AI couldn't make this profile, "failure" otherwise. */
  kind: "decline" | "failure";
  reason: string;
}

export type GenerationStatus = "idle" | "generating" | "done" | "error";

export interface IGenerationRequest {
  instructions: string;
  platform: AutofillConfigProfilePlatform;
  /** Left out for Unassigned and on Fleet Free. */
  fleetId?: number;
}

interface IGenerationState {
  status: GenerationStatus;
  /** Bumped by every run, so a result can't outlive the run it came from. */
  runId: number;
  readingDone: boolean;
  /** undefined while pending, null when the request failed. */
  nameDescription?: IAutofillConfigProfileNameDescriptionResponse | null;
  profile?: IAutofillConfigProfileContentResponse;
  error?: IGenerationError;
}

const INITIAL_STATE: IGenerationState = {
  status: "idle",
  runId: 0,
  readingDone: false,
};

/** The UI branches on the field the API reports the error on. */
export const toGenerationError = (e: unknown): IGenerationError => {
  const firstError = (e as AxiosResponse<IApiError> | undefined)?.data
    ?.errors?.[0];
  if (firstError?.name === "natural_language_instructions") {
    return { kind: "decline", reason: firstError.reason };
  }
  return { kind: "failure", reason: firstError?.reason ?? "" };
};

const isServerTimeout = (e: unknown) =>
  (e as AxiosResponse<IApiError> | undefined)?.data?.errors?.[0]?.reason ===
  SERVER_TIMEOUT_REASON;

interface IUseProfileGenerationOptions {
  /** Called when the cap is hit, after both requests are aborted. */
  onTimeout: () => void;
}

/** Runs the name and description request and the content request together
 * and tracks what the page shows while they're in flight. The content
 * response decides when the generation is done; the name never holds it up. */
const useProfileGeneration = ({ onTimeout }: IUseProfileGenerationOptions) => {
  const [state, setState] = useState<IGenerationState>(INITIAL_STATE);
  const runIdRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);
  const timersRef = useRef<ReturnType<typeof setTimeout>[]>([]);
  const onTimeoutRef = useRef(onTimeout);
  onTimeoutRef.current = onTimeout;

  const abortRun = useCallback(() => {
    controllerRef.current?.abort();
    controllerRef.current = null;
    timersRef.current.forEach(clearTimeout);
    timersRef.current = [];
  }, []);

  useEffect(() => abortRun, [abortRun]);

  const generate = useCallback(
    ({ instructions, platform, fleetId }: IGenerationRequest) => {
      abortRun();
      const controller = new AbortController();
      controllerRef.current = controller;
      const { signal } = controller;

      runIdRef.current += 1;
      const runId = runIdRef.current;
      setState({ ...INITIAL_STATE, status: "generating", runId });
      // Applies an update only while this run is still the current one.
      const update = (changes: Partial<IGenerationState>) => {
        if (signal.aborted) {
          return;
        }
        setState((prev) =>
          prev.runId === runId ? { ...prev, ...changes } : prev
        );
      };

      timersRef.current.push(
        setTimeout(() => update({ readingDone: true }), READING_STEP_MS)
      );
      const timeOut = () => {
        abortRun();
        setState({ ...INITIAL_STATE, runId });
        onTimeoutRef.current();
      };
      const capTimer = setTimeout(timeOut, GENERATION_TIMEOUT_MS);
      timersRef.current.push(capTimer);

      autofillAPI
        .getConfigProfileNameDescription({ instructions, platform, signal })
        .then((nameDescription) => update({ nameDescription }))
        .catch(() => update({ nameDescription: null }));

      autofillAPI
        .getConfigProfileContent({ instructions, platform, fleetId, signal })
        .then((profile) => {
          // The name request keeps running, so a late name can still fill
          // the summary.
          clearTimeout(capTimer);
          update({ status: "done", profile });
        })
        .catch((e: unknown) => {
          if (signal.aborted) {
            return;
          }
          if (isServerTimeout(e)) {
            timeOut();
            return;
          }
          // A failed profile discards any name and description shown.
          update({
            status: "error",
            error: toGenerationError(e),
            nameDescription: undefined,
          });
          abortRun();
        });
    },
    [abortRun]
  );

  /** Aborts both requests and returns to the instructions. */
  const stop = useCallback(() => {
    abortRun();
    setState({ ...INITIAL_STATE, runId: runIdRef.current });
  }, [abortRun]);

  let step: GenerationStep = "writing";
  if (!state.readingDone) {
    step = "reading";
  } else if (state.nameDescription === undefined) {
    step = "drafting";
  }

  return { ...state, step, generate, stop };
};

export default useProfileGeneration;
