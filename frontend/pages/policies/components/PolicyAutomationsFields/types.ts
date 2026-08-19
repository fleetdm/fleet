type IAutomationRowKey =
  | "ticket_webhook"
  | "install_software"
  | "run_script"
  | "calendar_event"
  | "conditional_access";

export interface IAutomationCheckboxRow {
  key: IAutomationRowKey;
  label: string;
  tooltip?: React.ReactNode;
  checked: boolean;
  onToggle: (next: boolean) => void;
  /** Feature isn't enabled for the fleet: greys the row, disables the checkbox,
   *  and shows the "Not enabled for <fleet>" hint. */
  isDisabled: boolean;
  /** Disables the checkbox (and greys the row) without showing any hint — used
   *  when the current user's role can't manage this automation. */
  isLocked?: boolean;
  picker?: React.ReactNode;
}
