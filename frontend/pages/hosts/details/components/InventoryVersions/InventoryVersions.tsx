import { uniq } from "lodash";
import React from "react";

import CopyButton from "components/buttons/CopyButton";
import Card from "components/Card";
import DataSet from "components/DataSet";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import TruncatedTextList from "components/TruncatedTextList";
import {
  SoftwareExtensionFor,
  formatSoftwareType,
  formatSoftwareVersion,
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  IHostSoftware,
  ISoftwareInstallVersion,
  SoftwareSource,
} from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { dateAgo } from "utilities/date_format";

const baseClass = "inventory-versions";

const fileName = (path: string | null) => path?.split("/").pop() || path;

/** How many of a keg's executables get a row of their own. A formula like
 * netpbm installs hundreds of tools, and a row each mounts a tooltip and a
 * copy button per executable — too many to scan, and all of them needed only
 * as a set, which the copy-all button hands over in one go. */
const MAX_EXECUTABLES_SHOWN = 10;

interface IInventoryVersionProps {
  version: ISoftwareInstallVersion;
  source: SoftwareSource;
  extension_for?: SoftwareExtensionFor;
  bundleIdentifier?: string;
}

const InventoryVersion = ({
  version,
  source,
  bundleIdentifier,
  extension_for,
}: IInventoryVersionProps) => {
  const {
    vulnerabilities,
    installed_paths: installedPaths,
    signature_information: signatureInformation,
  } = version;

  const lastOpenedTitle =
    INSTALLABLE_SOURCE_PLATFORM_CONVERSION[source] === "linux" ? (
      <TooltipWrapper tipContent="The last time the package was opened by the end user or accessed by any process on the host.">
        Last opened
      </TooltipWrapper>
    ) : (
      "Last opened"
    );

  return (
    <Card
      className={`${baseClass}__version`}
      color="grey"
      borderRadiusSize="medium"
    >
      <div className={`${baseClass}__row`}>
        <DataSet
          title="Version"
          value={
            formatSoftwareVersion({ ...version, source }) ||
            DEFAULT_EMPTY_CELL_VALUE
          }
          textOnly
        />
        <DataSet
          title="Type"
          value={formatSoftwareType({ source, extension_for })}
          textOnly
        />
        {bundleIdentifier && (
          <DataSet
            title="Bundle identifier"
            value={bundleIdentifier}
            textOnly
          />
        )}
        {version.last_opened_at !== undefined && (
          <DataSet
            title={lastOpenedTitle}
            value={
              version.last_opened_at !== ""
                ? dateAgo(version.last_opened_at)
                : "Never"
            }
            textOnly
          />
        )}
        {vulnerabilities && vulnerabilities.length !== 0 && (
          <DataSet
            className={`${baseClass}__vulnerabilities`}
            title="Vulnerabilities"
            value={<TruncatedTextList items={vulnerabilities} />}
          />
        )}
      </div>
      {!!installedPaths?.length &&
        installedPaths.map((path) => {
          // A path reports one signature entry per executable found under it,
          // so a Homebrew keg has as many entries as the formula has Mach-O
          // files while an app bundle has one.
          const pathSigInfo =
            signatureInformation?.filter(
              (info) => info.installed_path === path
            ) ?? [];
          const cdHash = pathSigInfo.find((info) => info.hash_sha256)
            ?.hash_sha256;
          const executables = pathSigInfo.filter(
            (info) => !info.hash_sha256 && info.executable_sha256
          );
          const hiddenExecutableCount = Math.max(
            executables.length - MAX_EXECUTABLES_SHOWN,
            0
          );
          // Hard links inside a keg share a hash, and a Santa rule needs each
          // hash once.
          const allExecutableHashes = uniq(
            executables.map((info) => info.executable_sha256)
          ).join("\n");

          return (
            <div className={`${baseClass}__sig-info`} key={path}>
              <DataSet orientation="horizontal" title="Path" value={path} />
              {cdHash && (
                <DataSet orientation="horizontal" title="Hash" value={cdHash} />
              )}
              {executables.slice(0, MAX_EXECUTABLES_SHOWN).map((info) => (
                <div
                  className={`${baseClass}__executable`}
                  key={`${info.executable_path}:${info.executable_sha256}`}
                >
                  <TooltipWrapper
                    className={`${baseClass}__executable-name`}
                    tipContent={info.executable_path}
                  >
                    {fileName(info.executable_path)}
                  </TooltipWrapper>
                  <span className={`${baseClass}__executable-hash`}>
                    {info.executable_sha256}
                  </span>
                  <CopyButton
                    copyText={info.executable_sha256 ?? ""}
                    variant="compact"
                    size="small"
                  />
                </div>
              ))}
              {executables.length > 1 && (
                <div className={`${baseClass}__executables-footer`}>
                  {hiddenExecutableCount > 0 && (
                    <span className={`${baseClass}__more-executables`}>
                      +{hiddenExecutableCount} more
                    </span>
                  )}
                  {/* compact, like the per-row buttons: subdued at size small
                  pads 8px on the right rather than 4px, which would leave this
                  icon short of the column the row icons form. */}
                  <CopyButton
                    copyText={allExecutableHashes}
                    ariaLabel="Copy all hashes"
                    variant="compact"
                    size="small"
                  >
                    Copy all hashes <Icon name="copy" size="small" />
                  </CopyButton>
                </div>
              )}
            </div>
          );
        })}
    </Card>
  );
};

interface IInventoryVersionsProps {
  hostSoftware: IHostSoftware;
  showLabel?: boolean;
}
const InventoryVersions = ({
  hostSoftware,
  showLabel = true,
}: IInventoryVersionsProps) => {
  const installedVersions = hostSoftware.installed_versions;

  if (!installedVersions || installedVersions.length === 0) {
    return (
      <div className={baseClass}>
        <Card
          className={`${baseClass}__version-details`}
          color="grey"
          borderRadiusSize="medium"
        >
          <div className={`${baseClass}__row`}>
            <DataSet
              title="Type"
              value={formatSoftwareType({
                source: hostSoftware.source,
                extension_for: hostSoftware.extension_for,
              })}
            />
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className={baseClass}>
      {showLabel && (
        <div className={`${baseClass}__label`}>
          Current version{installedVersions.length > 1 && "s"}:
        </div>
      )}
      <div className={`${baseClass}__versions`}>
        {installedVersions.map((installedVersion) => {
          return (
            <InventoryVersion
              key={`${installedVersion.version}|${installedVersion.release}`}
              version={installedVersion}
              source={hostSoftware.source}
              bundleIdentifier={hostSoftware.bundle_identifier}
              extension_for={hostSoftware.extension_for}
            />
          );
        })}
      </div>
    </div>
  );
};

export default InventoryVersions;
