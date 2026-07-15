/** Imperative handle each automation sub-form modal exposes so the parent
 *  AutomationsModal can validate and collect its data via a ref.
 *  `TData` is the shape returned by `getFormData`. */
export interface IAutomationFormHandle<TData> {
  /** The section's submit data, or null when it isn't configured / has
   *  nothing to submit. */
  getFormData: () => TData | null;
  /** Returns false (and surfaces field errors) when the form is invalid. */
  validate: () => boolean;
  /** Whether the form's values differ from what was initially loaded. */
  isDirty: () => boolean;
}
