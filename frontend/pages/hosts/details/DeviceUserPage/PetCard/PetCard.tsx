import React, { useContext, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import classnames from "classnames";

import deviceUserAPI from "services/entities/device_user";
import { NotificationContext } from "context/notification";
import {
  HostPetAction,
  HostPetMood,
  IHostPet,
} from "interfaces/host_pet";

import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "pet-card";

const PET_QUERY_KEY = (token: string) => ["device-pet", token] as const;

// Emoji for a given mood + species. Keeping species == "cat" only for now; easy
// to extend later by keying on species.
const MOOD_EMOJI: Record<HostPetMood, string> = {
  happy: "😺",
  content: "🐱",
  sad: "😿",
  hungry: "🙀",
  dirty: "🐈",
  sick: "😾",
};

const MOOD_CAPTION: Record<HostPetMood, string> = {
  happy: "is purring contentedly.",
  content: "is chilling.",
  sad: "looks glum — maybe play with them?",
  hungry: "is crying for food!",
  dirty: "is a grubby little goblin.",
  sick: "feels awful — try some medicine.",
};

interface IStatBarProps {
  label: string;
  value: number;
  // if true, lower values are "good" (like hunger)
  invertScale?: boolean;
  className?: string;
}

const StatBar = ({ label, value, invertScale, className }: IStatBarProps) => {
  // For inverted scales (hunger), we display 100-value so the bar reads "how
  // full/satisfied" rather than "how hungry". Keeps every bar "bigger = better".
  const displayValue = invertScale ? 100 - value : value;

  let color = "green";
  if (displayValue < 30) color = "red";
  else if (displayValue < 60) color = "yellow";

  return (
    <div className={classnames(`${baseClass}__stat`, className)}>
      <div className={`${baseClass}__stat-label`}>
        <span>{label}</span>
        <span className={`${baseClass}__stat-value`}>{displayValue}</span>
      </div>
      <div className={`${baseClass}__stat-track`}>
        <div
          className={classnames(
            `${baseClass}__stat-fill`,
            `${baseClass}__stat-fill--${color}`
          )}
          style={{ width: `${displayValue}%` }}
        />
      </div>
    </div>
  );
};

interface IPetCardProps {
  deviceAuthToken: string;
  className?: string;
}

const PetCard = ({ deviceAuthToken, className }: IPetCardProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const queryClient = useQueryClient();
  const [adoptName, setAdoptName] = useState("");

  const {
    data: petData,
    isLoading: isLoadingPet,
    isError: isErrorPet,
  } = useQuery(
    PET_QUERY_KEY(deviceAuthToken),
    () => deviceUserAPI.getDevicePet(deviceAuthToken),
    {
      enabled: !!deviceAuthToken,
      refetchOnMount: true,
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  const pet: IHostPet | null = petData?.pet ?? null;

  const invalidatePet = () =>
    queryClient.invalidateQueries(PET_QUERY_KEY(deviceAuthToken));

  const adoptMutation = useMutation(
    (name: string) =>
      deviceUserAPI.adoptDevicePet(deviceAuthToken, {
        name,
        species: "cat",
      }),
    {
      onSuccess: () => {
        renderFlash("success", "Welcome to the family!");
        setAdoptName("");
        invalidatePet();
      },
      onError: () =>
        renderFlash("error", "Could not adopt a pet. Please try again."),
    }
  );

  const actionMutation = useMutation(
    (action: HostPetAction) =>
      deviceUserAPI.applyDevicePetAction(deviceAuthToken, action),
    {
      onSuccess: () => invalidatePet(),
      onError: () =>
        renderFlash("error", "That didn't work. Try again in a sec."),
    }
  );

  if (isLoadingPet) {
    return (
      <div className={classnames(baseClass, className)}>
        <Spinner />
      </div>
    );
  }

  if (isErrorPet) {
    return (
      <div className={classnames(baseClass, className)}>
        <p>Could not load your pet right now. Please try again later.</p>
      </div>
    );
  }

  // Adoption flow.
  if (!pet) {
    return (
      <div
        className={classnames(baseClass, `${baseClass}--adopt`, className)}
      >
        <div className={`${baseClass}__adopt-art`}>🐾</div>
        <h2>Adopt a pet</h2>
        <p>
          Give your device a companion! The better you care for your device, the
          happier your pet will be.
        </p>
        <form
          className={`${baseClass}__adopt-form`}
          onSubmit={(e) => {
            e.preventDefault();
            const trimmed = adoptName.trim();
            if (trimmed.length === 0 || trimmed.length > 32) return;
            adoptMutation.mutate(trimmed);
          }}
        >
          <label htmlFor="pet-name">Name your cat</label>
          <input
            id="pet-name"
            type="text"
            maxLength={32}
            value={adoptName}
            onChange={(e) => setAdoptName(e.target.value)}
            placeholder="Whiskers"
          />
          <Button
            type="submit"
            disabled={
              adoptMutation.isLoading || adoptName.trim().length === 0
            }
          >
            {adoptMutation.isLoading ? "Adopting..." : "Adopt"}
          </Button>
        </form>
      </div>
    );
  }

  // Pet exists — show the tamagotchi UI.
  const emoji = MOOD_EMOJI[pet.mood] ?? MOOD_EMOJI.content;
  const caption = MOOD_CAPTION[pet.mood] ?? MOOD_CAPTION.content;

  const handleAction = (action: HostPetAction) => {
    if (actionMutation.isLoading) return;
    actionMutation.mutate(action);
  };

  return (
    <div className={classnames(baseClass, className)}>
      <div className={`${baseClass}__header`}>
        <h2>{pet.name}</h2>
        <span className={`${baseClass}__mood-pill`}>{pet.mood}</span>
      </div>

      <div
        className={classnames(
          `${baseClass}__stage`,
          `${baseClass}__stage--${pet.mood}`
        )}
      >
        <div className={`${baseClass}__pet-emoji`} aria-label={pet.mood}>
          {emoji}
        </div>
        <p className={`${baseClass}__caption`}>
          {pet.name} {caption}
        </p>
      </div>

      <div className={`${baseClass}__stats`}>
        <StatBar label="Health" value={pet.health} />
        <StatBar label="Happiness" value={pet.happiness} />
        <StatBar label="Fullness" value={pet.hunger} invertScale />
        <StatBar label="Cleanliness" value={pet.cleanliness} />
      </div>

      <div className={`${baseClass}__actions`}>
        <Button
          variant="default"
          onClick={() => handleAction("feed")}
          disabled={actionMutation.isLoading}
        >
          Feed
        </Button>
        <Button
          variant="default"
          onClick={() => handleAction("play")}
          disabled={actionMutation.isLoading}
        >
          Play
        </Button>
        <Button
          variant="default"
          onClick={() => handleAction("clean")}
          disabled={actionMutation.isLoading}
        >
          Clean
        </Button>
        <Button
          variant="default"
          onClick={() => handleAction("medicine")}
          disabled={actionMutation.isLoading || pet.health > 70}
        >
          Medicine
        </Button>
      </div>

      <p className={`${baseClass}__hint`}>
        Tip: the state of your device affects your pet too. Failing policies,
        disabled disk encryption, or a pet left alone will make them unwell.
      </p>
    </div>
  );
};

export default PetCard;
