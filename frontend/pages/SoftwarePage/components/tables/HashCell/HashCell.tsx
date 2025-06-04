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

// Utility to generate a random hex string of length 64 (SHA256-like)
const randomHash = () =>
  Array.from({ length: 64 }, () =>
    Math.floor(Math.random() * 16).toString(16)
  ).join("");

// Utility to generate 0, 1, or 2 hashes randomly
const randomHashes = () => {
  const count = Math.floor(Math.random() * 3); // 0, 1, or 2
  return Array.from({ length: count }, randomHash);
};

// Dummy data generator for ISoftwareInstallVersion[]
const generateDummyInstalledVersion = () => {
  // 50% chance of null, otherwise array of 1-2 items
  if (Math.random() < 0.5) return null;
  const numVersions = 1 + Math.floor(Math.random() * 2); // 1 or 2
  return Array.from({ length: numVersions }, () => ({
    signature_information: [
      {
        hash_sha256: randomHashes(),
      },
    ],
  }));
};

const HashCell = ({
  installedVersion,
  onClickMultipleHashes,
}: IHashCellProps) => {
  const installedVersion2 = generateDummyInstalledVersion();
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
    installedVersion2,
    (v) => v.signature_information ?? []
  );

  const allHash = flatMap(
    allSignatureInformation,
    (sigInfo) => sigInfo.hash_sha256 ?? []
  );
  const uniqueHash = new Set(allHash);
  const uniqueHashCount = uniqueHash.size;

  // 0 hashes
  if (installedVersion2 === null || uniqueHashCount === 0) {
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
