import React from "react";
import { useQuery } from "react-query";
import classnames from "classnames";
import { SOFTWARE_ICON_SIZES, SoftwareIconSizes } from "styles/var/icon_sizes";
import { AxiosError } from "axios";
import softwareAPI from "services/entities/software";
import { getMatchedSoftwareIcon } from "../";

const baseClass = "software-icon";

interface ISoftwareIconProps {
  /**  The software's name (used for fallback icon matching) */
  name?: string;
  /** The software's source (used for fallback icon matching) */
  source?: string;
  /** The icon size (default: 'small' 24x24 px) */
  size?: SoftwareIconSizes;
  /**  The image URL or API path to fetch custom icon blob */
  url?: string | null;
  /** Timestamp string of when the icon was last uploaded
   * (used to refetch stale icon if it was updated) */
  uploadedAt?: string;
}

const SoftwareIcon = ({
  name = "",
  source = "",
  size = "small",
  url,
  uploadedAt,
}: ISoftwareIconProps) => {
  const classNames = classnames(baseClass, `${baseClass}__${size}`);

  const isApiUrl =
    (typeof url === "string" && url?.startsWith("/api/")) || false;

  const { data: currentCustomIconBlob, isLoading } = useQuery<
    Blob | undefined,
    AxiosError,
    string
  >(
    ["softwareIcon", url, uploadedAt],
    () => softwareAPI.getSoftwareIconFromApiUrl(url as string),
    {
      enabled: isApiUrl,
      retry: false,
      select: (blob) => (blob ? URL.createObjectURL(blob) : ""),
    }
  );

  const imgClasses = classnames(
    `${baseClass}__software-img`,
    `${baseClass}__software-img-${size}`
  );

  let iconSrc: string | null = null;

  if (isApiUrl) {
    if (isLoading) {
      // Return empty div while loading custom icon so component size doesn't jump
      return <div className={classNames.concat(" loading-placeholder")} />;
    }
    if (currentCustomIconBlob) {
      // Uses custom icon blob from API if fetch succeeded
      iconSrc = currentCustomIconBlob;
    }
  } else if (url) {
    // Use direct image URL (e.g. VPP image URL))
    iconSrc = url;
  }

  if (iconSrc) {
    return (
      <div className={classNames}>
        <img className={imgClasses} src={iconSrc} alt="" />
      </div>
    );
  }

  // Fallback: Render a matched SVG icon by software name/source
  const MatchedIcon = getMatchedSoftwareIcon({ name, source });
  return (
    <MatchedIcon
      width={SOFTWARE_ICON_SIZES[size]}
      height={SOFTWARE_ICON_SIZES[size]}
      className={classNames}
    />
  );
};

export default SoftwareIcon;
