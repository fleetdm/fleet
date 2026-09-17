import classnames from "classnames";
import React from "react";

import CopyButton from "components/buttons/CopyButton";
import Card from "components/Card";
import DataSet from "components/DataSet";
import TooltipWrapper from "components/TooltipWrapper";
import TruncatedTextList from "components/TruncatedTextList";
import {
  SoftwareExtensionFor,
  formatSoftwareType,
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  IHostSoftware,
  ISoftwareInstallVersion,
  SoftwareSource,
} from "interfaces/software";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { dateAgo } from "utilities/date_format";

const baseClass = "inventory-versions";

const fileName = (path: string | null) => path?.split("/").pop() || path;

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
          value={version.version || DEFAULT_EMPTY_CELL_VALUE}
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

          return (
            <div
              className={classnames(`${baseClass}__sig-info`, {
                [`${baseClass}__sig-info--with-executables`]: !!executables.length,
              })}
              key={path}
            >
              <DataSet orientation="horizontal" title="Path" value={path} />
              {cdHash && (
                <DataSet orientation="horizontal" title="Hash" value={cdHash} />
              )}
              {executables.map((info) => (
                <DataSet
                  key={`${info.executable_path}:${info.executable_sha256}`}
                  orientation="horizontal"
                  title="Executable"
                  value={
                    <span className={`${baseClass}__executable`}>
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
                    </span>
                  }
                />
              ))}
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
              key={installedVersion.version}
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
