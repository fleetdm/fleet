import React from "react";
import { flatMap } from "lodash";

import Button from "components/buttons/Button";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipTruncatedTextCell from "components/TableContainer/DataTable/TooltipTruncatedTextCell";

import { ISoftwareInstallVersion } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "installed-path-cell";

interface IInstalledPathCellProps {
  installedVersion: ISoftwareInstallVersion[] | null;
  onClickMultiplePaths: () => void;
}

const InstalledPathCell = ({
  installedVersion,
  onClickMultiplePaths,
}: IInstalledPathCellProps) => {
  const allPaths = flatMap(installedVersion, (v) => v.installed_paths ?? []);
  const uniquePaths = new Set(allPaths);
  const uniquePathsCount = uniquePaths.size;

  if (installedVersion === null || uniquePathsCount === 0) {
    return (
      <TextCell className={baseClass} value={DEFAULT_EMPTY_CELL_VALUE} grey />
    );
  }

  if (uniquePathsCount === 1) {
    const [onlyPath] = uniquePaths;
    return <TooltipTruncatedTextCell className={baseClass} value={onlyPath} />;
  }

  // 2 or more installed versions
  return (
    <Button variant="text-link" onClick={onClickMultiplePaths}>
      {uniquePathsCount.toString()} paths
    </Button>
  );
};

export default InstalledPathCell;
