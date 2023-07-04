import { IconNames } from "components/icons";

export interface IEmptyTableProps {
  iconName?: IconNames;
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  className?: string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
}
