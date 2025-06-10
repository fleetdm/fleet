import React, { useState } from "react";
import { flatMap } from "lodash";
import { stringToClipboard } from "utilities/copy_text";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IHostSoftware, ISoftwareInstallVersion } from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "hash-cell";

interface IHashCellProps {
  installedVersion: ISoftwareInstallVersion[] | null;
  onClickMultipleHashes: (software: IHostSoftware) => void;
}

const HashCell = ({
  installedVersion,
  onClickMultipleHashes,
}: IHashCellProps) => {
  const [copyMessage, setCopyMessage] = useState("");

  const onCopySha256 = (hash: string) => (evt: React.MouseEvent) => {
    evt.preventDefault();

    stringToClipboard(hash)
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));

    // Clear message after 1 second
    setTimeout(() => setCopyMessage(""), 100);

    return false;
  };

  const renderHash = (hash: string) => {
    return (
      <>
        <span className={`${baseClass}__sha256`}>
          {hash.slice(0, 7)}&hellip;{" "}
        </span>
        <div className={`${baseClass}__sha-copy-button`}>
          <Button variant="icon" iconStroke onClick={onCopySha256(hash)}>
            <Icon name="copy" />
          </Button>
        </div>
        <div className={`${baseClass}__copy-overlay`}>
          {copyMessage && (
            <div
              className={`${baseClass}__copy-message`}
            >{`${copyMessage} `}</div>
          )}
        </div>
      </>
    );
  };

  const allSignatureInformation = flatMap(
    installedVersion,
    (v) => v.signature_information ?? []
  );

  const allHash = flatMap(
    allSignatureInformation,
    (sigInfo) => sigInfo.hash_sha256 ?? []
  );
  const uniqueHash = new Set(allHash);
  const uniqueHashCount = uniqueHash.size;

  // 0 hashes
  if (installedVersion === null || uniqueHashCount === 0) {
    return (
      <TextCell className={baseClass} value={DEFAULT_EMPTY_CELL_VALUE} grey />
    );
  }

  // 1 hash
  if (uniqueHashCount === 1) {
    const [onlyHash] = uniqueHash;
    return <div className={baseClass}>{renderHash(onlyHash)}</div>;
  }

  // 2 or more hashes
  return (
    <Button variant="text-link" onClick={onClickMultipleHashes}>
      {uniqueHashCount.toString()} hashes
    </Button>
  );
};

export default HashCell;
