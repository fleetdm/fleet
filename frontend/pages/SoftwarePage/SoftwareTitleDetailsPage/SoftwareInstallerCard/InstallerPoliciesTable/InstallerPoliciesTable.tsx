import React, { useCallback } from "react";
import classnames from "classnames";

import { ISoftwareInstallPolicy } from "interfaces/software";

import TableContainer from "components/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import CustomLink from "components/CustomLink";
import generateInstallerPoliciesTableConfig from "./InstallerPoliciesTableConfig";

const baseClass = "installer-policies-table";

interface IInstallerPoliciesTable {
  className?: string;
  teamId?: number;
  isLoading?: boolean;
  policies?: ISoftwareInstallPolicy[] | null;
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

  const renderTableHelpText = () => (
    <div>
      Software will be installed when hosts fail{" "}
      {policies?.length === 1 ? "this policy" : "any of these policies"}.{" "}
      <CustomLink
        url="https://fleetdm.com/learn-more-about/policy-automation-install-software"
        text="Learn more"
        newTab
      />
    </div>
  );

  return (
    <TableContainer
      className={baseClass}
      isLoading={isLoading}
      columnConfigs={softwareStatusHeaders}
      data={policies || []}
      renderCount={renderInstallerPoliciesCount}
      disablePagination
      disableMultiRowSelect
      emptyComponent={() => <></>}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      renderTableHelpText={renderTableHelpText}
    />
  );
};

export default InstallerPoliciesTable;
