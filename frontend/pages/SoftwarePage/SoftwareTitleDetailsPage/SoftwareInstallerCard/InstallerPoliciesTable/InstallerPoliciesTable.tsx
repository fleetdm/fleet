import React, { useCallback } from "react";
import classnames from "classnames";

import { ISoftwareInstallPolicyUI } from "interfaces/software";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import generateInstallerPoliciesTableConfig from "./InstallerPoliciesTableConfig";

export const baseClass = "installer-policies-table";

interface IInstallerPoliciesTable {
  className?: string;
  teamId?: number;
  isLoading?: boolean;
  policies?: ISoftwareInstallPolicyUI[] | null;
}
const InstallerPoliciesTable = ({
  className,
  teamId,
  isLoading = false,
  policies,
}: IInstallerPoliciesTable) => {
  const classNames = classnames(baseClass, className);

  const softwareStatusHeaders = generateInstallerPoliciesTableConfig({
    teamId,
  });

  const renderInstallerPoliciesCount = useCallback(() => {
    return <TableCount name="policies" count={policies?.length} />;
  }, [policies?.length]);

  return (
    <TableContainer
      className={classNames}
      isLoading={isLoading}
      columnConfigs={softwareStatusHeaders}
      data={policies || []}
      renderCount={renderInstallerPoliciesCount}
      disablePagination
      disableMultiRowSelect
      emptyComponent={() => <></>}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      hideFooter
    />
  );
};

export default InstallerPoliciesTable;
