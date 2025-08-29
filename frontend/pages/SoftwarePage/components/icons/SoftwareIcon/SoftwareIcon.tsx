import React from "react";
import { useQuery } from "react-query";
import classnames from "classnames";
import { SOFTWARE_ICON_SIZES, SoftwareIconSizes } from "styles/var/icon_sizes";
import { AxiosError } from "axios";
import softwareAPI from "services/entities/software";
import { getMatchedSoftwareIcon } from "../";

const baseClass = "software-icon";

// Extracts softwareId and teamId from API url, e.g. "/api/latest/fleet/software/titles/90/icon?team_id=2"
const extractParams = (url: string) => {
  const [path, queryString = ""] = url.split("?");
  const params = new URLSearchParams(queryString);
  const pathSegments = path.split("/");
  const idSegment = pathSegments[pathSegments.length - 2];
  const softwareId =
    idSegment && !isNaN(Number(idSegment)) ? Number(idSegment) : undefined;
  const teamIdParam = params.get("team_id");
  const teamId =
    teamIdParam && !isNaN(Number(teamIdParam))
      ? Number(teamIdParam)
      : undefined;
  return { softwareId, teamId };
};

interface ISoftwareIconProps {
  name?: string;
  source?: string;
  size?: SoftwareIconSizes;
  /** Accepts an image url to display for the software icon image. */
  url?: string | null;
}

const SoftwareIcon = ({
  name = "",
  source = "",
  size = "small",
  url,
}: ISoftwareIconProps) => {
  const classNames = classnames(baseClass, `${baseClass}__${size}`);

  const isApiUrl = url?.startsWith("/api/");
  let softwareId: number | undefined;
  let teamId: number | undefined;
  if (isApiUrl && url) {
    ({ softwareId, teamId } = extractParams(url));
  }

  // Only run useQuery if both IDs are numbers
  const shouldFetch =
    isApiUrl && typeof softwareId === "number" && typeof teamId === "number";

  const { data: iconBlob } = useQuery<Blob | undefined, AxiosError, string>(
    ["softwareIcon", softwareId, teamId],
    () => softwareAPI.getSoftwareIcon(softwareId as number, teamId as number), // safe to assert here
    {
      enabled: shouldFetch,
      retry: false,
      select: (blob) => (blob ? URL.createObjectURL(blob) : ""),
    }
  );

  let iconSrc: string | null = null;
  if (isApiUrl && iconBlob) {
    iconSrc = iconBlob;
  } else if (url) {
    iconSrc = url;
  }

  if (iconSrc) {
    const imgClasses = classnames(
      `${baseClass}__software-img`,
      `${baseClass}__software-img-${size}`
    );
    return (
      <div className={classNames}>
        <img className={imgClasses} src={iconSrc} alt="" />
      </div>
    );
  }

  const MatchedIcon = getMatchedSoftwareIcon({ name, source });
  return (
    <MatchedIcon
      width={SOFTWARE_ICON_SIZES[size]}
      height={SOFTWARE_ICON_SIZES[size]}
      viewBox="0 0 32 32"
      className={classNames}
    />
  );
};

export default SoftwareIcon;
