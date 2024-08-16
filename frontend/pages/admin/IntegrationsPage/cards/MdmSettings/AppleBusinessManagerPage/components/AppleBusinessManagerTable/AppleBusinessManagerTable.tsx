import { IMdmAbmToken } from "interfaces/mdm";
import React from "react";

const baseClass = "apple-business-manager-table";

interface IAppleBusinessManagerTableProps {
  abmTokens: IMdmAbmToken[];
}

const AppleBusinessManagerTable = ({
  abmTokens,
}: IAppleBusinessManagerTableProps) => {
  return <div className={baseClass}>abm table</div>;
};

export default AppleBusinessManagerTable;
