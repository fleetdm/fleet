import { GraphicNames } from "components/graphics";

export interface IEmptyTableProps {
  graphicName?: GraphicNames;
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  className?: string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
}
