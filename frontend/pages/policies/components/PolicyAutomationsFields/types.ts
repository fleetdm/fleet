export type TicketOrWebhookState = "webhook" | "ticket" | "disabled";

type IAutomationRowKey =
  | "ticket_webhook"
  | "install_software"
  | "run_script"
  | "calendar_event"
  | "conditional_access";

export interface IAutomationRow {
  key: IAutomationRowKey;
  label: string;
  tooltip?: React.ReactNode;
  checked: boolean;
  onToggle: (next: boolean) => void;
  isDisabled: boolean;
  picker?: React.ReactNode;
}
