import React from "react";

import { IGeoLocation } from "interfaces/host";

import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import LastUpdatedText from "components/LastUpdatedText";

import { HostMdmDeviceStatusUIState } from "../../helpers";

const baseClass = "location-modal";

export const getCityCountryLocation = (hostGeolocation: IGeoLocation) =>
  [hostGeolocation.city_name, hostGeolocation.country_iso]
    .filter(Boolean)
    .join(", ");

interface IIosOrIpadosDetails {
  isIosOrIpadosHost: boolean;
  hostMdmDeviceStatus: HostMdmDeviceStatusUIState;
}

const getLocationMessage = (
  iosOrIpadosDetails: IIosOrIpadosDetails | undefined,
  hasLocation: boolean
): JSX.Element | null => {
  const FETCH_LATEST_LOCATION_MESSAGE = (
    <>
      Close this modal and select <strong>Refetch</strong> to fetch new
      location.
    </>
  );
  const IOS_LOCK_REQUIRED_MESSAGE = (
    <>
      To view location, Apple requires that iOS hosts are locked (Lost Mode)
      first.
    </>
  );

  if (!iosOrIpadosDetails?.isIosOrIpadosHost) {
    return FETCH_LATEST_LOCATION_MESSAGE;
  }

  const { hostMdmDeviceStatus } = iosOrIpadosDetails;

  switch (hostMdmDeviceStatus) {
    case "unlocked":
      return <div>{IOS_LOCK_REQUIRED_MESSAGE}</div>;

    case "locking":
      return (
        <>
          <p>{IOS_LOCK_REQUIRED_MESSAGE}</p>
          <p>
            Lock is pending. Host will lock the next time it checks in to Fleet.
          </p>
        </>
      );

    case "locating":
      return (
        <div>
          Location is pending. Host will share location the next time it checks
          in to Fleet.
        </div>
      );

    default:
      return (
        <div>
          {!hasLocation ? "Location not available. " : ""}
          {FETCH_LATEST_LOCATION_MESSAGE}
        </div>
      );
  }
};

const buildGoogleMapsLinkFromGeo = (loc: IGeoLocation): string | null => {
  const { country_iso, city_name, geometry } = loc;

  // Prefer coordinates if valid
  if (
    geometry &&
    Array.isArray(geometry.coordinates) &&
    geometry.coordinates.length >= 2
  ) {
    const [lat, lng] = geometry.coordinates; // GeoJSON is [lat, lng]
    if (lat != null && lng != null) {
      return `https://www.google.com/maps?q=${lat},${lng}`;
    }
  }

  // Fallback to "City, Country"
  if (city_name && country_iso) {
    const query = encodeURIComponent(`${city_name}, ${country_iso}`);
    return `https://www.google.com/maps/search/?api=1&query=${query}`;
  }

  return null;
};

interface ILocationModal {
  hostGeolocation?: IGeoLocation;
  onExit: () => void;
  onClickLock: () => void;
  iosOrIpadosDetails?: IIosOrIpadosDetails;
  detailsUpdatedAt?: string;
}

const LocationModal = ({
  hostGeolocation,
  onExit,
  onClickLock,
  iosOrIpadosDetails,
  detailsUpdatedAt,
}: ILocationModal) => {
  console.log("hostGeolocation:", hostGeolocation);
  console.log("iosOrIpadosDetails:", iosOrIpadosDetails);
  const googleMapsUrl = hostGeolocation
    ? buildGoogleMapsLinkFromGeo(hostGeolocation)
    : null;

  const isIosOrIpadosHost = iosOrIpadosDetails?.isIosOrIpadosHost || false;
  const isIosLockedWithLocationAvail =
    iosOrIpadosDetails?.isIosOrIpadosHost &&
    iosOrIpadosDetails?.hostMdmDeviceStatus === "locked" &&
    hostGeolocation !== null;

  const shouldShowLastUpdatedAt =
    !isIosOrIpadosHost || isIosLockedWithLocationAvail;

  const renderLastUpdatedAt = () => (
    <LastUpdatedText
      lastUpdatedAt={detailsUpdatedAt}
      customTooltipText="The last time location data was updated."
    />
  );

  const renderContent = () => {
    return (
      <>
        {hostGeolocation && (
          <div className={`${baseClass}__location`}>
            {hostGeolocation && getCityCountryLocation(hostGeolocation)}{" "}
            {googleMapsUrl && (
              <div className={`${baseClass}__link`}>
                <CustomLink
                  url={googleMapsUrl}
                  text="Open in Google Maps"
                  newTab
                  multiline
                />
              </div>
            )}
          </div>
        )}
        <div className={`${baseClass}__message`}>
          {getLocationMessage(iosOrIpadosDetails, Boolean(hostGeolocation))}
          {shouldShowLastUpdatedAt && renderLastUpdatedAt()}
        </div>
      </>
    );
  };

  const renderFooter = () => {
    const isIos = iosOrIpadosDetails?.isIosOrIpadosHost;
    const status = iosOrIpadosDetails?.hostMdmDeviceStatus;

    if (isIos && status === "unlocked") {
      return (
        <ModalFooter
          primaryButtons={
            <>
              <Button type="button" onClick={onExit} variant="inverse">
                Cancel
              </Button>
              <Button type="button" onClick={onClickLock}>
                Lock
              </Button>
            </>
          }
        />
      );
    }

    return (
      <ModalFooter
        primaryButtons={
          <Button type="button" onClick={onExit}>
            Done
          </Button>
        }
      />
    );
  };

  return (
    <Modal title="Location" className={baseClass} onExit={onExit}>
      <>
        {renderContent()}
        {renderFooter()}
      </>
    </Modal>
  );
};

export default LocationModal;
